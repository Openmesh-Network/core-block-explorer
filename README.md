# Core block explorer

A simple block explorer currently being used for testing.
It displays all the network info we need for debugging.

Currenly very ugly. 
Will have to improve UX/UI in the near future to use in actual production.

What it does is it subscribes to the cometbft api and listens to new block events.
When triggered, it renders each block into an html page and stores it on disk.
On requests it just loads the html file from disk.
