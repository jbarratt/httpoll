# httpoll

It was a dark and stormy night. You were about to make some (totally safe!) change to some http-serving system. But you just wanted that extra safety blanket to make sure it wasn't all going to explode as you worked.

Enter httpoll.

You ran

    $ httpoll http://domain1.com/foo http://domain2.com https://domain3.com

and were able to work in peace, a beautiful realtime feed of poll times scrolling away in your terminal.

![Screenshot](http://drop.serialized.net/httpoll.png)

# TODO

- [ ] refactor, document, test
- [ ] support http-less options
- [ ] fail elegantly when no options provided
- [ ] handle ctrl-c as well as q
- [x] get & time hardcoded domain & exit
- [x] get multiple domains in goroutines
- [x] grok termui basics
- [x] figure out data structure needed to store results over time
- [x] initialize and put git repo

