# Installing PostgreSQL on OpenWrt

NOTE: If the postgres server has already been setup but broken, you can try to remove the data directory and the system will re-initialize it on next start.

```sh
rm -rf /srv/postgresql/data
```

Otherwise, follow these steps to install and set up PostgreSQL on OpenWrt:

1. requierment:

make sure you can run these command on your shell
adduser , deluser, addgroup, delgroup, su

2. install packages

```sh
opkg update
opkg install pgsql-server pgsql-cli
```

3. change the path of datebase and log file.

```sh
uci set postgresql.config.PGDATA=/srv/postgresql/data

uci set postgresql.config.PGLOG=/srv/postgresql/data/postgresql.log

uci commit
```

4. initial databse

```sh
mkdir -p /srv/postgresql/data

chown postgres /srv/postgresql/data

sudo -u  postgres

$LC_COLLATE="C" initdb --pwprompt -D /srv/postgresql/data
```

when the command finish, follow the output to start database

```sh
pg_ctl -D /srv/postgresql/data -l logfile start
```
