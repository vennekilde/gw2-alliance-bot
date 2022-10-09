# gw2-alliance-bot

A Guild Wars 2 bot that can add guild roles and verification roles based on the user's linked Guild Wars 2 account

Utilizes the [gw2verify](https://github.com/vennekilde/gw2verify) backend, originally designed for WvW world verification. 

## How it works

Currently, it works by looking for roles named [<GUILD_TAG>] <GUILD_NAME> and allows the user to join the role, if they are a member of the guild in the role.
Additionally, if the user is able to join any guild role on the server, the bot will also try and assign the role "API Verified", in order to make permission management centralized.

In the future, these should be converted to managed roles by the bot and created by the menu to be provided when typing /setup

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

## Commands

### /verify

![verify command preview](https://i.imgur.com/HturIae.png)

![insert api key preview](https://i.imgur.com/T8onxLb.png)

![verify success & guild role auto assignment](https://i.imgur.com/gZjhJfQ.png)

### /rep

![pick guild to represent, if more than one](https://i.imgur.com/svCFNEn.png)

![guild role auto assignment](https://i.imgur.com/oLxf6lX.png)

![no guild role available for the user](https://i.imgur.com/se1Ygma.png)

### /status

![status command preview](https://i.imgur.com/FMQuk3E.png)