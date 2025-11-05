# Installing Mariadb On OpenWrt

Install `mariadb-server` package:

```
opkg update && opkg install mariadb-server
```

Add these lines to the bottom of `/etc/mysql/my.cnf`:

```
[mysqld]
datadir = /srv/mysql
tmpdir  = /tmp
```

Create mysql data dir:

```
mkdir -p /srv/mysql
chown -R mariadb:mariadb /srv/mysql
```

Enable mysql service in `/etc/config/mysqld`:

```
config mysqld 'general'

    # Unless enable, MariaDB will not start without this
        option enabled '1'
```

Run mysql install script by command:

```
mysql_install_db --force
```

Fix permission errors:

```
chown -R mariadb:mariadb /srv/mysql
```

Start and enable `mysqld` service:

```
service mysqld start
service mysqld enable
```

Set `root` password:

```
mysqladmin -u root password 'root-password-here'
```
