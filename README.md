# Term GPT
A bubbletea tui app for chat GPT

This is a proof of concept at best

## TODO
- [ ] save/resume chats (sqlite probably)
    - ui mostly implemented
- [ ] help modal for controls
- [ ] implement a markdown bubble for the chat responses
- [ ] need some better styling
- [ ] new config options
    - [ ] max tokens to use per request
    - [ ] max number of prev messages to send along with requests
- [x] The spinner does not spin when waiting for a response
- [x] app does not take over the whole terminal window
- [x] dynamically resize viewport or terminal resize
    - its not perfect but it will do for now

## Known issues
- When GPT is thinking you can still see the top row of the textarea
- there is a lot of flickering on ui update
- on sending a request the chat viewport gets messed up
