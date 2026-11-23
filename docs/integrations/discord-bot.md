# Discord bot: setup, launch and verification

## Architecture

Discord is the primary user interface for the first release. The bot connects to Discord Gateway and registers guild-scoped slash commands. It communicates with the backend over the private Docker Compose network.

The backend accepts bot requests only with `DISCORD_BOT_API_TOKEN`. This credential is separate from `FOUNDRY_SYNC_API_TOKEN`; never reuse either token. Discord user ID, guild ID and member role IDs are read by the bot from the authenticated Gateway interaction. The backend independently checks the configured guild, registered account and application permission.

New members can register themselves with `/register` and receive the `player` role. A configured trusted Discord user ID receives the `owner` role on registration. Application roles live in PostgreSQL. Discord role mappings are evaluated on every request, so a mapped role can grant permissions without making Discord the application's only authorization source.

## 1. Create the Discord application and bot

1. Open [Discord Developer Portal](https://discord.com/developers/applications) and create an application, for example `FoundryMarches`.
2. On **General Information**, copy **Application ID**. This is `DISCORD_APPLICATION_ID`.
3. Open **Bot** and add a bot if Discord has not created one automatically.
4. Reset/copy the bot token. This is `DISCORD_BOT_TOKEN`. Store it only in the server's local `.env`; never commit it or send it in chat.
5. Privileged Gateway intents (Presence, Server Members, Message Content) are not required for the current commands. The bot uses the Guilds intent and interaction payloads.
6. Under **OAuth2 → URL Generator**, select scopes `bot` and `applications.commands`. Grant the bot **View Channels** and **Send Messages**. Open the generated URL and add the bot to the campaign server.

## 2. Obtain the server ID and trusted admin ID

1. In the Discord client, open **User Settings → Advanced** and enable **Developer Mode**.
2. Right-click the one server where the bot will operate and choose **Copy Server ID**. This is `DISCORD_GUILD_ID`.
3. Right-click your own Discord profile and choose **Copy User ID**. Put this ID into `DISCORD_ADMIN_USER_IDS`. Multiple trusted IDs may be comma-separated.
4. The bot registers commands only for the configured guild. The backend rejects requests for any other guild.

## 3. Configure the application

On the server, edit the existing `.env` (do not replace it with `.env.example`). Add:

```dotenv
DISCORD_BOT_TOKEN=the-token-from-the-Developer-Portal
DISCORD_APPLICATION_ID=the-application-id
DISCORD_GUILD_ID=the-server-id
DISCORD_ADMIN_USER_IDS=your-discord-user-id
DISCORD_BOT_API_TOKEN=generated-random-secret
DISCORD_ADMIN_CHANNEL_ID=discord-admin-channel-id
# Optional; default 60 seconds, allowed range 15..3600
DISCORD_HEALTH_INTERVAL_SECONDS=60
```

Generate the service-to-service token independently:

```sh
openssl rand -hex 32
```

Copy its output into `DISCORD_BOT_API_TOKEN`. Do not use the Discord bot token or Foundry sync token for this value.

## 4. Apply the database migration

This is an existing database migration, not a fresh-volume bootstrap. Run it once against the existing `foundry_marches` database:

```sh
docker compose exec -T postgres psql -X -v ON_ERROR_STOP=1 \
  -U foundry_admin -d foundry_marches \
  < docs/database/migrations/0002_discord_bot_auth.sql
```

The migration is safe to re-run. It seeds permissions for the existing `player`, `gm`, `admin` and `owner` roles and grants the API database role the minimum additional access needed for registration, role assignment, role mapping, draft character creation and audit inserts. It does not delete or recreate data and does not touch the PostgreSQL volume.

## 5. Build and start

From the repository root:

```sh
docker compose config --quiet
docker compose up -d --build api discord-bot
docker compose ps
```

Inspect startup logs:

```sh
docker compose logs --tail=100 api
docker compose logs --tail=100 discord-bot
```

The API must be healthy. The bot log should report that it connected and registered commands for the configured guild. If the bot token, application ID or guild ID is wrong, the bot container will exit and report the Discord API error.

## 6. First registration and permissions

1. In the configured Discord server, run `/register` as the trusted admin account.
2. The backend creates/updates the Discord account, grants `player`, and grants `owner` because the Discord ID is in `DISCORD_ADMIN_USER_IDS`.
3. Run `/foundry-actors` and `/characters`. The trusted owner should be able to use both because the owner role has `foundry.manage`.
4. Have another server member run `/register`. They receive only `player`; they cannot list or link Foundry Actors.
5. As owner, run `/role-grant discord_user_id:<ID> role_key:gm` to grant the application GM role to a registered member. Alternatively, use `/role-map discord_role_id:<ID> role_key:gm` to map a Discord role to the application GM role. The mapping is evaluated using the member roles included in each Gateway interaction.
6. Only `player` and `gm` may be granted/mapped with these commands. The privileged `owner` role is only bootstrapped from `DISCORD_ADMIN_USER_IDS`.

## 7. Test draft creation and linking with Андрей

The bot now supports two explicit GM actions:

- `/character-draft` creates an application character in `draft` status from a selected unlinked Foundry Actor. It copies only the Actor name and portrait URL. It does not assign an owner or copy game statistics/items. Repeating the command for the same Actor returns the existing draft instead of creating a duplicate.
- `/character-link` lets the GM select an unlinked Foundry Actor and an existing application character, then preview and confirm the link. The confirmation is bound to the initiating Discord user and guild.

After linking, the existing Foundry sync path fills Foundry-owned character fields. Foundry sync itself still does not create application characters.

1. Ensure the Foundry world is synchronized after any PF2e system update. If needed, run the existing sync endpoint using the existing `FOUNDRY_SYNC_API_TOKEN`:
   `POST http://127.0.0.1:8080/api/integrations/foundry/sync`.
2. In Discord, run `/foundry-actors` and confirm Андрей's Actor is present.
3. Run `/character-draft`. Select Андрей in the dropdown. The bot should report a new `draft` character and its application ID. No owner is assigned.
4. Run `/characters`. Confirm the new Андрей draft appears.
5. Run `/character-link`. Select Андрей's Foundry Actor, then select the newly created Андрей application character.
6. Review the ephemeral confirmation message and click **Подтвердить привязку**.
7. Run `/foundry-actors` again. Андрей's Actor should no longer be in the unlinked list.
8. Run the Foundry sync again. Verify that Foundry-owned fields are populated from the Actor snapshot.
9. Verify the relation with a read-only SQL query:

```sql
SELECT fa.actor_uuid, fa.character_id, c.name, c.status,
       c.owner_user_id, fa.owner_confirmed_at, fa.linked_at
FROM foundry_actors fa
JOIN characters c ON c.id = fa.character_id
WHERE c.name ILIKE '%Андрей%';
```

The draft creation operation writes one `characters` row and one audit event. The subsequent link operation changes the Foundry link and link timestamps and adds its audit event. Neither operation sets `characters.owner_user_id` or `foundry_actors.owner_confirmed_at`, and neither writes to Foundry.

### If Андрей is missing

- Missing from `/foundry-actors`: run Foundry sync and check that the Actor exists in the configured world and has type `character`, not `npc`.
- Missing from `/characters`: create the draft explicitly with `/character-draft`; Foundry sync intentionally does not create application characters.
- The Actor or application character is already linked: the backend rejects the conflicting link with HTTP 409.
- Permission denied: register first and ensure the user has the application `gm`, `admin` or `owner` permission that includes `foundry.manage`.

## 8. Character lookup and integration monitoring

The `/character name:<имя>` command searches the latest Actor snapshots stored in PostgreSQL. It does not call Relay or Foundry for each lookup. If multiple Actors match, choose the correct one from the ephemeral select menu. The resulting Discord embed shows only values present in the decoded PF2e DTO: level, class/build references, HP, XP, explicitly stored ability scores/modifiers, skill ranks, hero points and a short inventory list. Missing source values are shown as unavailable rather than inferred.

For the initial release, `/character` requires the existing `foundry.manage` permission (GM/admin/owner). It is intentionally not exposed to ordinary players until character ownership and player-facing visibility rules are implemented.

The `/integration-status` command performs an on-demand check of PostgreSQL, the Relay HTTP service and the Foundry connection. When `DISCORD_ADMIN_CHANNEL_ID` is configured, the bot also runs the same check periodically (60 seconds by default) and posts a message only when one or more component states change. Relay reachability is tested separately from a real authenticated Foundry Actor search through Relay.

The admin channel receives:
- integration outage and recovery notifications;
- backend network errors and HTTP 5xx responses encountered by bot commands;
- repeated identical errors are rate-limited to one notification per five minutes.

This is operational alerting, not a replacement for container logs: detailed stack traces and failures from background processes that do not pass through a bot request remain available through `docker compose logs`.

To configure the channel, enable Developer Mode in Discord, copy the target text channel ID, set `DISCORD_ADMIN_CHANNEL_ID` in the existing server `.env`, and ensure the bot can view the channel and send messages with embeds. Rebuild the bot after changing the environment:

```sh
docker compose up -d --build api discord-bot
docker compose logs --tail=100 discord-bot
```

Then test:
1. Run `/character name:Андрей` as a GM/admin and select Андрей.
2. Run `/integration-status`; verify PostgreSQL, Relay and Foundry are green.
3. Do not deliberately disconnect the production Relay or Foundry session just to test alerts. Verify recovery notifications during a planned maintenance window or in a separate staging environment.

## Current command reference

| Command | Purpose | Required application permission |
|---|---|---|
| `/register` | Register the current server member | Server membership |
| `/foundry-actors` | List unlinked Foundry character Actors | `foundry.manage` |
| `/characters` | List application characters and IDs | `foundry.manage` |
| `/character-draft` | Create a draft from a selected Foundry Actor | `foundry.manage` |
| `/character-link` | Select Actor and application character, preview and confirm a link | `foundry.manage` |
| `/character` | Search saved Foundry snapshots and display a character embed | `foundry.manage` |
| `/integration-status` | Check PostgreSQL, Relay and Foundry connectivity | Registered server member |
| `/role-grant` | Grant `player` or `gm` | `settings.manage` |
| `/role-map` | Map a Discord role to `player` or `gm` | `settings.manage` |

## Operational notes

- API and bot communicate through the Compose network; the bot does not need the public API port.
- Keep the API and bot service tokens distinct. Rotate `DISCORD_BOT_API_TOKEN` in both services together.
- The pending confirmation button expires after 10 minutes and is bound to the Discord user and guild that initiated it. A bot restart invalidates pending confirmations.
- Do not grant privileged Discord roles to untrusted users. A role mapping intentionally grants the corresponding application role to every member who has that Discord role.
- Select menus show up to 25 entries per page; navigation buttons paginate longer Actor/character lists. Selection flows expire after 10 minutes and are bound to the initiating Discord user and guild. A bot restart invalidates them.
- The current implementation covers registration, RBAC checks, role assignment/mapping, explicit draft creation, Foundry Actor linking, GM-only character lookup and integration health alerts. Web OAuth2/browser sessions, player-facing character visibility, ownership confirmation and character approval remain separate work.
