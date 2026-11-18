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

The migration is safe to re-run. It seeds permissions for the existing `player`, `gm`, `admin` and `owner` roles and grants the API database role the minimum additional access needed for registration, role assignment, role mapping and audit inserts. It does not delete or recreate data and does not touch the PostgreSQL volume.

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

## 7. Test the link flow with Андрей

The link operation deliberately connects an already-existing application character to an already-discovered Foundry Actor. It does not create an application character, infer ownership, or write anything to Foundry.

1. Ensure the Foundry world is synchronized after any PF2e system update. If needed, run the existing sync endpoint using the existing `FOUNDRY_SYNC_API_TOKEN`:
   `POST http://127.0.0.1:8080/api/integrations/foundry/sync`.
2. In Discord, run `/foundry-actors` and find Андрей's Actor. Copy its exact Actor UUID.
3. Run `/characters` and find the existing application character Андрей. Copy its exact application character ID.
4. Run `/character-link actor_uuid:<Actor UUID> character_id:<character ID>`.
5. Review the ephemeral confirmation message. It shows both names and identifiers. Click **Подтвердить привязку**.
6. Run `/foundry-actors` again. Андрей's Actor should no longer be in the unlinked list.
7. Verify the relation with a read-only SQL query:

```sql
SELECT fa.actor_uuid, fa.character_id, c.name, c.owner_user_id,
       fa.owner_confirmed_at, fa.linked_at
FROM foundry_actors fa
JOIN characters c ON c.id = fa.character_id
WHERE c.name ILIKE '%Андрей%';
```

The operation changes only `foundry_actors.character_id` and link timestamps and adds an audit event. It does not set `characters.owner_user_id` or `foundry_actors.owner_confirmed_at`.

### If Андрей is missing

- Missing from `/foundry-actors`: run Foundry sync and check that the Actor exists in the configured world and has type `character`, not `npc`.
- Missing from `/characters`: there is not yet an application character record to link. Foundry sync intentionally does not create application characters. Do not insert a row manually into production just for this test; the character creation/approval flow is a separate application feature.
- The Actor or application character is already linked: the backend rejects the conflicting link with HTTP 409.
- Permission denied: register first and ensure the user has the application `gm`, `admin` or `owner` permission that includes `foundry.manage`.

## Current command reference

| Command | Purpose | Required application permission |
|---|---|---|
| `/register` | Register the current server member | Server membership |
| `/foundry-actors` | List unlinked Foundry character Actors | `foundry.manage` |
| `/characters` | List application characters and IDs | `foundry.manage` |
| `/character-link` | Preview and confirm a link | `foundry.manage` |
| `/role-grant` | Grant `player` or `gm` | `settings.manage` |
| `/role-map` | Map a Discord role to `player` or `gm` | `settings.manage` |

## Operational notes

- API and bot communicate through the Compose network; the bot does not need the public API port.
- Keep the API and bot service tokens distinct. Rotate `DISCORD_BOT_API_TOKEN` in both services together.
- The pending confirmation button expires after 10 minutes and is bound to the Discord user and guild that initiated it. A bot restart invalidates pending confirmations.
- Do not grant privileged Discord roles to untrusted users. A role mapping intentionally grants the corresponding application role to every member who has that Discord role.
- The current implementation covers registration, RBAC checks, role assignment/mapping and explicit Foundry Actor linking. Web OAuth2/browser sessions, character creation, ownership confirmation and character approval remain separate work.
