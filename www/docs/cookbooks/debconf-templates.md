# Use debconf and templates

Deb installation format has a support for user input during installation using [debconf](https://manpages.debian.org/testing/debconf-doc/debconf-devel.7.en.html).

To enable it inside `goreleaser` you need:

`templates` file, what to ask (all templates go into single file):

```none
Template: foo/like_debian
Type: boolean
Description: Do you like Debian?
 We'd like to know if you like the Debian GNU/Linux system.

Template: foo/why_debian_is_great
Type: note
Description: Poor misguided one. Why are you installing this package?
 Debian is great. As you continue using Debian, we hope you will
 discover the error in your ways.
```

Maintainer script file that will trigger questions, usually its `postinst` because all package files are already installed:

```sh
#!/bin/sh -e

# Source debconf library.
. /usr/share/debconf/confmodule

# Do you like debian?
db_input high foo/like_debian || true
db_go || true

# Check their answer.
# with db_get you load value into $RET env variable.
db_get foo/like_debian
if [ "$RET" = "false" ]; then
    # Poor misguided one...
    db_input high foo/why_debian_is_great || true
    db_go || true
fi
```

Include `templates` and `postinst` in `.goreleaser.yml`:

```yaml
    overrides:
      deb:
        scripts:
          postinstall: ./deb/postinst
    deb:
      scripts:
        templates: ./deb/templates
```

Useful tutorial: [Debconf Programmer's Tutorial](http://www.fifi.org/doc/debconf-doc/tutorial.html)
