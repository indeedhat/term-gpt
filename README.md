# Term GPT
A bubbletea tui app for chat GPT

This app is still a bit rough arround the edges (to say the least) but it is now in a useable state.

## Controls
- Press tab to toggle between the chat history view and the chat textarea
- j,k/up,down can be used to scroll the chat history
- Ctrl+c to exit
- Ctrl+n scrolls the chat window down
- Ctrl+p scrolls the chat window up

## TODO
- [ ] help modal for controls
- [ ] need some better styling
- [ ] save/resume chats (sqlite probably)
    - [x] save chats
    - [x] display list of chat history
    - [x] switch to previous chats
    - [ ] delete previous chats
- [x] implement a markdown bubble for the chat responses
- [x] new config options
    - [x] max tokens to use per request
    - [x] max number of prev messages to send along with requests
- [x] The spinner does not spin when waiting for a response
- [x] app does not take over the whole terminal window
- [x] dynamically resize viewport or terminal resize
    - its not perfect but it will do for now

## Known issues
- [ ] when scrolling up in the chat viewport only the top half of the viewport scrolls until the next textarea blink event
- [ ] there is a lot of flickering on ui update
    - this only appears to happen in some termitals and not others, in alacritty it only happens when running via tmux
- [x] on sending a request the chat viewport gets messed up
- [x] When GPT is thinking you can still see the top row of the textarea
- [x] resizing the window can break the ui cutting off the top border of the chat/history viewports
