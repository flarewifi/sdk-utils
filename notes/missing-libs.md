When building on OpenWRT x64 and -buildmode=plugin, you will encounter an error like:

```
gcc failed exit status 1
/usr/bin/ld: cannot find -lpthread
/usr/bin/ld: cannot find -lresolv
/usr/bin/ld: cannot find -ldl
```

To fix this error, you need to create a dummy archive files like so:

```
ar -rc /usr/lib/libpthread.a
ar -rc /usr/lib/libresolv.a
ar -rc /usr/lib/libdl.a
```
