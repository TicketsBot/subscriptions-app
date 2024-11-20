# subscriptions-app
`subscriptions-app` is a Discord app (as opposed to bot) that provides a slash command to look up a user's subscription
status on Patreon via their email address.

![Example Screenshot](/docs/img/example.png)

## Usage
Some experience with Discord app development is assumed.

1. Set up a new app on the [developer portal](https://discord.dev).
2. Run the slash command creation script using `go run cmd/createcommands/main.go -token <bot token>`.
3. Set up a [Patreon app](https://www.patreon.com/portal/registration/register-clients).
4. Run the main binary: there are 2 ways of doing this - either by building and running the main binary directly
   (`go build cmd/app/main.go`), or via Docker (recommended). If running the binary directly, see the
   [envvars.md](/envvars.md) file for a list of environment variables that need to be set. 


   Alternatively, a `config.json` file can be used to configure the application. See the
   [config.json.example](/config.json.example) file for an example.

Note, anyone is able to use the command, as long as the command is run in a guild listed in the `DISCORD_ALLOWED_GUILDS`
environment variable. You should use Discord's built-in application command permission system to restrict usage to
trusted users only.

## Running via Docker
1. Go to the [GitHub Packages page](https://github.com/TicketsBot/subscriptions-app/pkgs/container/subscriptions-app) to
find the latest image, and pull it:
```shell
docker pull ghcr.io/ticketsbot/subscriptions-app:COMMIT_HASH_HERE
```

2. Copy the example `.env.example` file to `.env` and fill in the values.

3. Run the Docker container!
```shell
docker run -d \
    --env-file=.env \
    -p 8080:8080 \
    --restart=always \
    ghcr.io/ticketsbot/subscriptions-app:COMMIT_HASH_HERE
```

4. Set up a reverse proxy with HTTPS to the container. The app listens on port 8080 by default. Then, submit the URL
`https://<your domain>/interaction` to Discord as the interaction endpoint URL.