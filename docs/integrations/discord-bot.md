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
6. Under **OAuth2 → URL Generator**, select scopes `bot` and `applications.commands`. Grant the bot **View Channels**, **Send Messages** and **Manage Roles**. The campaign access role must be below the bot's highest role in the server role hierarchy. Open the generated URL and add the bot to the campaign server.

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
# Discord role that grants access to campaign channels
DISCORD_CAMPAIGN_ROLE_ID=1574386309584388106
# Optional; default 60 seconds, allowed range 15..3600
DISCORD_HEALTH_INTERVAL_SECONDS=60
# Public Foundry URL used to resolve relative Actor portrait paths in Discord embeds
FOUNDRY_PUBLIC_BASE_URL=https://foundry.example.com
# Automatic Foundry world snapshot refresh interval (30..86400 seconds; default 300)
FOUNDRY_SYNC_INTERVAL_SECONDS=300
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

The migration is safe to re-run. It seeds permissions for the existing `player`, `gm`, `admin` and `owner` roles and grants the API database role the minimum additional access needed for registration, role assignment, role mapping, draft character creation, ownership-scoped character reads and audit inserts. It does not delete or recreate data and does not touch the PostgreSQL volume.

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
3. After the database registration succeeds, the bot grants the Discord role configured by `DISCORD_CAMPAIGN_ROLE_ID` (default `1574386309584388106`). This role is for access to campaign channels; it is separate from application RBAC roles. If Discord role assignment fails, the account remains registered; fix the bot permission/role hierarchy and run `/register` again to retry the idempotent role grant.
4. Run `/foundry-actors` and `/characters`. The trusted owner should be able to use both because the owner role has `foundry.manage`.
5. Have another server member run `/register`. They receive the application `player` role and the Discord campaign-access role; they cannot list or link Foundry Actors.
5. As owner, run `/role-grant discord_user_id:<ID> role_key:gm` to grant the application GM role to a registered member. Alternatively, use `/role-map discord_role_id:<ID> role_key:gm` to map a Discord role to the application GM role. The mapping is evaluated using the member roles included in each Gateway interaction.
6. Only `player` and `gm` may be granted/mapped with these commands. The privileged `owner` role is only bootstrapped from `DISCORD_ADMIN_USER_IDS`.

## 7. Test draft creation and linking with Андрей

Character creation and linking support both players and GMs:

- `/character-create name:<имя>` lets a player create an application draft owned by their own registered account.
- A GM uses `/character-create name:<имя> owner:@игрок`. The selected player must already have run `/register`; the backend resolves the mention to that player's internal account and stores explicit ownership.
- `/character-draft` remains available to GMs who want to create a draft from a selected unlinked Foundry Actor. The draft is initially unowned.
- `/character-link owner:@игрок` lets a GM select an unlinked Foundry Actor and an application character, then preview and confirm the link. The owner mention is mandatory for every link; if the character already has a different owner, the backend rejects the operation rather than silently transferring it.
- On link confirmation, the backend atomically links the Actor, assigns the selected owner if the character was unowned, records the ownership transfer/audit context, and marks ownership confirmed for that link. A conflicting existing owner is never silently replaced.
- A player-created draft is already assigned to that player; when a GM links its Foundry Actor, the existing owner is preserved and confirmed.
- The confirmation is bound to the initiating Discord user and guild.

After linking, the existing Foundry sync path fills Foundry-owned character fields. Foundry sync itself still does not create application characters.

1. Ensure the Foundry world is synchronized after any PF2e system update. If needed, run the existing sync endpoint using the existing `FOUNDRY_SYNC_API_TOKEN`:
   `POST http://127.0.0.1:8080/api/integrations/foundry/sync`.
2. In Discord, run `/foundry-actors` and confirm Андрей's Actor is present.
3. Run `/character-draft`. Select Андрей in the dropdown. The bot creates a `draft`; it has no owner until the link operation.
4. Run `/characters`. Confirm the new Андрей draft appears.
5. Run `/character-link owner:@игрок`. Mention the Discord account that will own Андрей, then select Андрей's Foundry Actor and the application character. The mentioned player must have completed `/register`.
6. Review the ephemeral confirmation message. Verify the displayed owner mention, then click **Подтвердить привязку**.
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

The draft creation operation writes one `characters` row and one audit event. The link operation atomically associates the Foundry Actor, assigns the explicitly mentioned owner if the character is still unowned, records the transfer when needed, marks `owner_confirmed_at`, and adds its audit event. Neither operation writes to Foundry.

### If Андрей is missing

- Missing from `/foundry-actors`: run Foundry sync and check that the Actor exists in the configured world and has type `character`, not `npc`.
- Missing from `/characters`: create the draft explicitly with `/character-draft`; Foundry sync intentionally does not create application characters.
- The Actor or application character is already linked: the backend rejects the conflicting link with HTTP 409.
- Permission denied: register first and ensure the user has the application `gm`, `admin` or `owner` permission that includes `foundry.manage`.

## 8. Character lookup and integration monitoring

The /character name:<имя> command searches the latest Actor snapshots stored in PostgreSQL. It does not call Relay or Foundry for each lookup. If exactly one Actor matches, the bot opens its card immediately; if multiple Actors match, choose the correct one from the ephemeral select menu. The same single-result shortcut is used by /my-character and /character-unlink. Unlinking always requires an explicit confirmation button.

The main character card is intentionally compact and contains the portrait, class, ancestry, level, current/max HP, six ability scores/modifiers and XP. Buttons below the card open separate views for abilities and skills, equipment, spells and campaign metrics. These views replace the contents of the same Discord message rather than creating a message for each group. The campaign-metrics button currently reports that a campaign metrics data source is not yet connected.

If hp.max is absent, the decoder derives it from class HP per level, ancestry HP, character level and Constitution modifier. The Constitution modifier is read from prepared Actor abilities when available; otherwise it is reconstructed only when the required ancestry/background/class selections and level-based boosts are present. If the inputs are incomplete, maximum HP remains unknown rather than being guessed. The class key ability is taken from the class Item system.keyAbility, which takes precedence over the Actor-level details.keyability value.

The post-session regression fixture at src/backend/internal/foundrysync/testdata/actor-after-session.json is a complete Actor export with 46 Items, level 2 and 400 XP. It verifies the Investigator key ability is INT and derives maximum HP 24. Its system.abilities field is null, so the six ability scores remain unknown: saved build boosts and ancestry selections are not substituted for Actor-prepared ability values or passed to the derived-stat calculator. The fixture also has no system.attributes.dying; the decoder leaves that condition unknown when absent. When a dying value and maximum are present and the current dying value reaches that maximum, the Discord card is marked МЁРТВ.

Relative portrait paths are resolved against FOUNDRY_PUBLIC_BASE_URL, and the main card uses the resolved URL as its image. Set the variable to the externally reachable Foundry base URL, not the Relay URL. Discord must be able to fetch the resulting image URL without an interactive Foundry login. If an image still does not render, open the exact resolved URL in a private/incognito browser window; a login page, 403 or 404 means the image is not publicly fetchable by Discord. Absolute HTTP(S) image URLs work without this setting.

The abilities button separates PF2e documents by their source type/category:
- class abilities (feat with system.category=class);
- ancestry abilities (feat with system.category=ancestry);
- actions (type=action), with the action kind shown where available;
- class features from the class progression table, limited to the character's current level and deduplicated against matching Actor Items;
- other feats (for example, general and skill feats);
- trained skills with their proficiency ranks.

The equipment button displays all physical Actor Items grouped by type (weapons, armor, shields, equipment, consumables, containers and treasure). Item state is read from equipped.carryType, equipped.inSlot and equipped.invested: only an armor item actually in its armor slot is labeled as worn armor; invested items are marked invested; stowed items inside a backpack are marked as stored in the backpack. A generic carryType=worn value is not by itself rendered as “worn”.

The spells button groups spell Items inside one Discord message by tradition, rank and spellcasting entry/list. Each group is a field in the same embed. Very large collections are shortened only when Discord's message/embed limits require it.

Access is enforced by the backend, not only by Discord command visibility:
- GM/admin/owner users with foundry.manage can search all synchronized character Actors.
- A registered player with characters.create can search and open only their own draft, pending_approval or active character, whose linked foundry_actors.owner_confirmed_at is set after the latest ownership transfer (or after the original link if there was no transfer).
- Unlinked, unconfirmed, unavailable, dead, retired, NPC, archived and other players' characters are not returned to players. Direct requests with another Actor UUID are also rejected.

Character ownership is assigned explicitly during /character-create or /character-link. The backend does not infer ownership from Foundry Actor names, Discord nicknames or other indirect signals. A linked character becomes visible to its assigned player only after the explicit owner assignment is committed and owner_confirmed_at is recorded for that link.

The /integration-status command performs an on-demand check of PostgreSQL, the Relay HTTP service and the Foundry connection. When DISCORD_ADMIN_CHANNEL_ID is configured, the bot also runs the same check periodically (60 seconds by default) and posts a message only when one or more component states change. Relay reachability is tested separately from a real authenticated Foundry Actor search through Relay.

The admin channel receives:
- integration outage and recovery notifications;
- backend network errors and HTTP 5xx responses encountered by bot commands;
- repeated identical errors are rate-limited to one notification per five minutes.

This is operational alerting, not a replacement for container logs: detailed stack traces and failures from background processes that do not pass through a bot request remain available through docker compose logs.

To configure the channel, enable Developer Mode in Discord, copy the target text channel ID, set DISCORD_ADMIN_CHANNEL_ID in the existing server .env, and ensure the bot can view the channel and send messages with embeds. Rebuild the bot after changing the environment.

Then test:
1. Run /character name:Андрей as a GM/admin; a unique match should open the compact card immediately. Use the buttons to inspect abilities, equipment and spells without creating extra messages.
2. Run /my-character as a player with a linked, owner-confirmed character in draft, pending_approval or active; only that player's character should be returned. A single result should open automatically.
3. Run /character-inventory name:<имя> and /character-spells name:<имя>; verify that all item/spell groups appear as fields in one Discord message each.
4. Search for another player's character as the same player; it must not appear. A direct request using another Actor UUID must also return 404.
5. If no ownership has been assigned and confirmed yet, an ordinary player should receive no character results. Do not change production ownership fields solely to make this test pass.
6. Run /integration-status; verify PostgreSQL, Relay and Foundry are green.
7. Do not deliberately disconnect the production Relay or Foundry session just to test alerts. Verify recovery notifications during a planned maintenance window or in a separate staging environment.

## Current command reference

| Command | Purpose | Required application permission |
|---|---|---|
| `/register` | Register the current server member | Server membership |
| `/foundry-actors` | List unlinked Foundry character Actors | `foundry.manage` |
| `/characters` | Show a readable list of application characters | `foundry.manage` |
| `/character-create` | Create a draft for yourself, or as GM for a mentioned player | `characters.create` for self; `foundry.manage` for another player |
| `/character-draft` | Create a draft from a selected Foundry Actor | `foundry.manage` |
| `/character-link owner:@player` | Select Actor and application character, assign/confirm owner, preview and confirm link | `foundry.manage` |
| `/character-unlink name:<имя> [owner:@player]` | Select an Actor, then confirm the unlink. Optionally explicitly reassign the application character owner. Without owner, the current owner is preserved. Transfer history is retained. | `foundry.manage` |
| `/foundry-sync` | Refresh all Actor snapshots immediately | `foundry.manage` |
| `/character` | Search and display character data; GM sees all, players see only their own linked and owner-confirmed drafts, pending approvals and active characters | `characters.create` for own characters; `foundry.manage` for global access |
| `/my-character` | Show the caller's own linked and owner-confirmed character(s) | `characters.create` or `foundry.manage` |
| `/character-inventory name:<имя>` | Show all physical Actor Items, grouped by type, with equipment/container state | Same visibility as `/character` |
| `/character-spells name:<имя>` | Show all spell Items grouped by tradition, rank and spellcasting entry | Same visibility as `/character` |
| `/integration-status` | Check PostgreSQL, Relay and Foundry connectivity | Registered server member |
| `/role-grant` | Grant `player` or `gm` | `settings.manage` |
| `/role-map` | Map a Discord role to `player` or `gm` | `settings.manage` |

## Operational notes

- API and bot communicate through the Compose network; the bot does not need the public API port.
- Keep the API and bot service tokens distinct. Rotate `DISCORD_BOT_API_TOKEN` in both services together.
- Link and unlink confirmation buttons expire after 10 minutes and are bound to the Discord user and guild that initiated them. A bot restart invalidates pending confirmations.
- Do not grant privileged Discord roles to untrusted users. A role mapping intentionally grants the corresponding application role to every member who has that Discord role.
- Select menus show up to 25 entries per page; navigation buttons paginate longer Actor/character lists. Selection flows expire after 10 minutes and are bound to the initiating Discord user and guild. A bot restart invalidates them.
- The current implementation covers registration, automatic campaign Discord-role assignment, RBAC checks, role assignment/mapping, player/GM character draft creation, explicit owner assignment during Foundry linking, owner-aware character directory output, personal character lookup, categorized character cards, inventory and spell views, confirmed Actor unlinking/explicit owner reassignment with audit history, startup/manual/periodic Foundry sync, and integration health alerts. Maximum HP is derived only when all required source inputs are present; missing inputs remain explicitly unknown. Web OAuth2/browser sessions and character approval remain separate work.
