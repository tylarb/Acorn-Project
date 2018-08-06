## Acorn-Project
Acorn is a slackbot which links resources and support channels to relevant keywords


# Usage

Using Acorn is easy! Simply send a message in any channel or private message with the bot, alerting the bot with it's name, and it will search the database for relevant information: 

![alt text](https://github.com/Tylarb/Acorn-Project/blob/master/screenshots/acorn_summary.png "Usage")


The component channels and playbooks should be populated and maintained by the product owner, but users can add tags by simply marking the appropriate channel with new tags:

![alt text](https://github.com/Tylarb/Acorn-Project/blob/master/screenshots/add_tag.png "New Tag")

The tag will immediately show up in future queries: 

![alt text](https://github.com/Tylarb/Acorn-Project/blob/master/screenshots/new_tag_display.png "Display new tag")


Of course, a help message is available just by typing "help" or "@Acorn help":


![alt text](https://github.com/Tylarb/Acorn-Project/blob/master/screenshots/acorn_help.png "Help")


# Behind the scenes

Acorn is written completely in Golang and runs on Pivotal Cloud Foundy. [nlopes' slack](https://github.com/nlopes/slack) is used to interface with the Slack API.

A Postgres database is used for backing storage, but all tags are loaded into an in-memory cache at application start to avoid database calls in general usage. This greatly improves performance.

Fuzzy logic for keyword matching, using the [levenshtein distance](github.com/texttheater/golang-levenshtein/levenshtein), allows the bot to handle mispellings of keywords. 


# Contributting 

Feel free to fork and submit pull requests, or submit issues and feature requests. 
