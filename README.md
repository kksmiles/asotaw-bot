# asotaw-bot

## Available commands

You must be in a voice channel to use the bot.
1. Type ```!gosing {youtube_url}``` to add a song to the queue
2. Type ```!gosing-queue``` to display the queue
3. Type ```!gosing-leave``` to clear the queue and leave the voice channel

## Up Next

1. Fix a bug where the bot plays the last song of the queue twice
2. ```!gosing-skip``` to skip the current song
3. ```!gosing-pause``` to pause the current song
4. ```!gosing-resume``` to resume the current song
5. ```!gosing-loop``` to loop the queue
6. ```!gosing-loop-one``` to loop the current song
7. ```!gosing-clear``` to clear the queue
8. ```!gosing-help``` to display the help message
9. Update the bot to support multiple guilds.
10. Add a simple text-based database to store the queue of each guild.

## How to setup the bot

# Step 1 : Create a discord application & bot

1. Go to https://discord.com/developers/applications
2. Click on "New Application"
3. Give it a name
4. Select your newly created application
5. Click on "Bot" in the left menu
6. Click on "Add Bot"
7. Click on "Copy" under "Token" and save it somewhere

# Step 2 : Invite your bot to your server

1. Go to Oauth2 > URL Generator (https://discord.com/developers/applications/{yourappid}/oauth2/url-generator)
2. Select "bot" in the "scopes" section
3. Select the permissions you want your bot to have in the "bot permissions" section
4. The required permissions are "Send Messages", "Read Message History", "Manage Messages", "Add Reactions", "Connect", "Speak", "Use Voice Activity"
5. Copy the generated link and paste it in your browser
6. Select the server you want to invite your bot to
7. Click on "Continue"
8. Click on "Authorize"

# Step 3 : Create a youtube application & get a token

1. Go to https://console.developers.google.com/
2. Click on "Create Project"
3. Give it a name
4. Click on "Create"
5. Click on "Enable APIs and Services"
6. Search for "Youtube Data API v3"
7. Click on "Enable"
8. Click on "Create Credentials"
9. Choose "API key"
10. Copy the generated token and save it somewhere

# Step 4 : Run the bot

1. Clone this repository
2. Use docker to build the image
``` docker build -t asotaw-bot . ```
3. Run the image
``` docker run -d -e "DISCORD_TOKEN={Refer to step 1}" -e "YOUTUBE_TOKEN={Refer to step 3}" --name {container_name} asotaw-bot ```
4. Enjoy