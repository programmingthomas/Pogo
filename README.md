#Pogo v0.1

Podcasts + Go = Pogo

Pogo is designed to be a self-hosted (if not entirely local) podcast catcher that works in your browser. The catcher will automatically download and check for new podcast episodes in the background so that you can listen or watch them straight from your browser.

##Progress
The basic UI has been completed and it will currently download audio/video podcasts quite happily. I still need to work on a lot of the bugs, improve speed, sort out the various concurrency issues and various other things that will improve it significantly. I would really appreciate some feedback...

##Installation
Either use 'go get github.com/programmingthomas/Pogo' or git clone to download this repo to your Go path before building the entire directory with 'go build'. Then execute pogo, which will start a server on [localhost](http://localhost:8888) which you should then open in your browser.

Currently the project has no dependencies however I plan to use SQLite in the future for data storage.

##License
Apache License, see LICENSE file for more info.