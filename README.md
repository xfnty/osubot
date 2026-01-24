**osubot** is a minimal Osu! IRC bot that creates and manages a single multiplayer room.

It is packaged as a single executable file that doesn't require any other software to be installed, like
NodeJS or .NET runtime.

If the bot suddenly exits, check `crash.txt` and restart it so it would rejoin the lobby. After that you will
have to define the queue by hand using `!q names...`. Otherwise host rotation will be disabled.

## Commands

The bot supports most of the commands players would expect it to and a few others related to its settings.

| Command            | Description                                                      | Access      |
| :----------------- | :-----------------------------------------------------------     | :---------- |
| `!q [names...]`    | Prints the host queue or defines it if executed by the owner.    | Anyone      |
| `!tl`, `!timeleft` | Prints estimated time left until the end of the match.           | Anyone      |
| `!m`, `!mirrors`   | Prints links to download mirrors for the current beatmap.        | Anyone      |
| `!s`, `!skip`      | Transfers host to the next player in the queue.                  | Host, Owner |
| `!hr [on/off]`     | Enabled/disables host rotation or prints its status.             | Owner       |
| `!dc [on/off]`     | Enabled/disables difficulty constraint or prints its status.     | Owner       |
| `!dcr min max`     | Defines difficulty constraint range or prints it out.            | Owner       |

The `names` in `!q` command are approximations if players' nicknames written as one or many of their first
letters in lowercase. If username contains whitespace, use double quotes: `"a player"`. Players not in the
`names` list will be added to the end of the queue in random order. For example, `!q mr m` will match `mrekk`
and `milosz`.

Difficulty constraint will not work until a beatmap that matches the constraint range is selected.

If you want me to add more commands, [send me an email][email] or [open an issue][issue]. Also pull requests
are always welcome).

## Setting Up

The bot doesn't require much setup except saving player's Osu! Web and IRC API credentials into `config.json`
file. To get those credentials, go to [Profile Settings][settings] and create a new IRC password and OAuth
application (callback URL can be empty). Then simply run the `.exe` and enter them.

The `config.json` file has the following structure:
```json
{
    "irc": {
        "address": "irc.ppy.sh:6667",
        "username": "username",
        "password": "01234567",
        "rate_limit": 4
    },
    "api": {
        "address": "https://osu.ppy.sh",
        "id": "12345",
        "secret": "abcd1234"
    },
    "host_rotation": {
        "enabled": true,
        "print_queue": true
    },
    "diffuclty_constraint": {
        "enabled": false,
        "range": [0, 10]
    }
}
```

`host_rotation.print_queue` flag will make the bot print the host queue every time the match finishes. Set it
to `false` if you want the bot to be more quiet.

[email]: mailto:xfnty.x@gmail.com
[issue]: https://github.com/xfnty/osubot/issues/new
[settings]: https://osu.ppy.sh/home/account/edit#legacy-api
