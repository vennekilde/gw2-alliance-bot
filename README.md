# gw2-alliance-bot

A Guild Wars 2 bot that can add guild roles and verification roles based on the user's linked Guild Wars 2 account

Utilizes the [gw2verify](https://github.com/vennekilde/gw2verify) backend, originally designed for WvW world verification. 

## Features

### World vs World Server Verification

**Notice:** Since servers are being removed in favor of dynamic worlds with world restructuring, it is likely this feature will stop working until the ArenaNet Guilds Wars 2 API is updated.

If the bot is configured to verify users against a server, the bot will assign configured roles to users based on the server they are linked to in Guild Wars 2.

#### Configuring

use `/settings` to configure which server the bot should verify users against and which role should be assigned to verified users.

World selection:

![guild role auto assignment](https://i.imgur.com/PKKpcVa.png)

Role assignment:

![guild role auto assignment](https://i.imgur.com/88C4N50.png)

### Guild Role Assignment

The bot will automatically assign roles to users based on the guilds they are a member of. The roles must be named in the format of `[{GUILD_TAG}] {GUILD_NAME}`.

As an example, if a user is a member of the guild with the tag `[TEST]` and the name `Test Guild`, the bot will assign the role `[TEST] Test Guild` to the user, if the role exists on the discord server.

### Guild Verification Role Assignment

To make it easier to manage permissions, the bot can be configured to assign a role of your choice to all users that are in a guild that has a role on the server.

If the bot is configured to assign the role `Verified` to all users with a guild role on the discord server, a user who is a member of the guild e.g. `[PYRE] Cinder Ashes` and that role is configured on the discord server, the bot will assign the role `Verified` to the user.

#### Configuring

use `/settings` to configure the role that should be assigned to users with guild roles on the server.

![guild role auto assignment](https://i.imgur.com/bEClidh.png)

## Commands

### /verify

Link a Guild Wars 2 account to your Discord account. This will allow the bot to fetch your account information from the Guild Wars 2 API.

The bot will ask you to insert your API key with a <ins>**specific**</ins> apikey name in order to prevent abuse. You can create an API key on the [Guild Wars 2 website](https://account.arena.net/applications).

You can link multiple Guild Wars 2 accounts to your Discord account. 

Use `/status` to see which accounts are linked to your Discord account.

![verify command preview](https://i.imgur.com/5l5wMKF.png)

### /status

Get the current status of your linked Guild Wars 2 accounts.

![status command preview](https://i.imgur.com/UAw105y.png)

### /refresh

Refresh your linked Guild Wars 2 account. This will force the bot to re-fetch your account information from the Guild Wars 2 API.

### /rep

Pick a guild to represent. This will present you with a list of all guilds you are a member of, that have a role on the server.

![pick guild to represent, if more than one](https://i.imgur.com/svCFNEn.png)

## Building

### Docker Image

`make package`

The code will be compiled during the docker build process

### Target: Host Machine

`make build`

### Target: Linux

`make build_linux`

### Target: Windows

`make build_windows`
