# matebook-applet
System tray applet/control app for Huawei Matebook

[![Build Status](https://travis-ci.com/nekr0z/matebook-applet.svg?branch=master)](https://travis-ci.com/nekr0z/matebook-applet) [![codecov](https://codecov.io/gh/nekr0z/matebook-applet/branch/master/graph/badge.svg)](https://codecov.io/gh/nekr0z/matebook-applet) [![Go Report Card](https://goreportcard.com/badge/github.com/nekr0z/matebook-applet)](https://goreportcard.com/report/github.com/nekr0z/matebook-applet)

---

## WARNING

Due to [a recent Codecov security problem](https://about.codecov.io/security-update/) the PGP key that was used to sign binaries and the Debian repository could have been compromised. It is stronly recommended to remove the old key from your trusted `apt` keyring since it can no longer really be trusted:
```
sudo apt-key del F25E85CB21A79726
```
The repository uses a new key now, you can add it [by usual means](#debian).

---

This simple applet is designed to make some of the proprietary Huawei PC Manager's functionality available via GUI on Linux.
 
![Applet screenshot](screenshot-applet.png?raw=true "matebook-applet")

It can be used as a system tray applet (above) or as a windowed app (below).

![Windowed screenshot](screenshot-windowed.png?raw=true "matebook-applet windowed mode")

##### Table of Contents
* [Installation and setup](#installation-and-setup)
  * [Debian and derivatives](#debian)
  * [Other Linux](#other-linux) - kernel 5.0 and newer
  * [Old Linux](#old-linux) - for those with pre-5.0 kernel
  * [Compiling](#compiling-matebook-applet) by yourself
* [Usage](#usage)
  * [Gnome](#gnome)
* [Development](#development)
  * [Contributing translations](#contributing-translations)
* [Credits](#credits)

#### Help `matebook-applet` get better!
Join the [development](#development) (or just [buy me a coffee](https://www.buymeacoffee.com/nekr0z), that helps, too).

## Installation and setup
The applet requires no installation as such. However, it has dependencies, its functionality may be limited without proper setup, and you may want to add it to autorun so that it gets started automatically when you load your DE. Read on for the details.

### Debian
Debian users can add this repo:
```
deb [signed-by=/usr/share/keyrings/matebook-applet.key] http://evgenykuznetsov.org/repo/ buster main
```
to their `/etc/apt/sources.list` and add the public key:
```
$ wget -qO - https://raw.githubusercontent.com/nekr0z/matebook-applet/master/matebook-applet.key | sudo tee /usr/share/keyrings/matebook-applet.key
```

From there, the applet is just a `sudo apt install matebook-applet` away.

Users report that this way also works on Debian derivatives such as Ubuntu or Linux Mint.

<details>
<summary><b>note to pre-2022 users</b></summary>

The repository used to be signed by other public keys, and this README used to recommend adding the key to the system-wide trusted keyring. This practice is no longer recommended by Debian because the key is then trusted by the system to sign any known repository, which is a security weakness. If you installed matebook-applet `.deb` package via this repository before June 2021, you may still have the old keys in your system-wide trusted keyring, and it's a good idea to remove them:
```
sudo apt-key del 0BA9571368CD3C34EFCA7D40466F4F38E60211B0
sudo apt-key del F25E85CB21A79726
sudo apt-key del FA32B7DDA1A3AC2C
```

</details>

### Other Linux
If you're running a Linux with relatively new kernel version, it already includes the required driver. Make sure your system has GTK+ and `libappindicator` installed, download (from the [releases page](https://github.com/nekr0z/matebook-applet/releases)) or [build](#compiling-matebook-applet) `matebook-applet`, and it should work *in read-only mode* right away.

For pre-5.5 kernels you may need to update the [Huawei-WMI driver](https://github.com/aymanbagabas/Huawei-WMI). The applet requres at least version 3.0 of the driver.

To be able *to change settings* as opposed to simply displaying them, you either need to run the applet as root (absolutely not recommended), or make sure all the necessary files (the hooks in `/sys/devices/platform/huawei-wmi` as well as `/etc/default/huawei-wmi/` directory) are user-writable. A good way to set everything up is to make use of [Rouven Spreckels](https://github.com/n3vu0r)' awesome [project](https://github.com/qu1x/huawei-wmi):
```
$ git clone https://github.com/qu1x/huawei-wmi.git
$ cd huawei-wmi/generic
$ sudo make install
```
You may need to re-login for adding your user to group to take effect.

### Old Linux

On Linux kernels earlier than 5.0 the Huawei-WMI driver is not available, so the alternative method should be used.

<details>
<summary><b>details</b></summary>

You can make `matebook-applet` work using the `-r` command line option and two scripts as explained below. Any one of the scripts is enough, but you need both for full functionality. You may download them both in one archive [here](https://github.com/nekr0z/linux-on-huawei-matebook-13-2019/releases). Please note that these scripts are for MateBook 13 and will not work on other MateBooks (you can still get them to work by changing them to address EC registers proper for your particular model, but make sure you know exactly what you're doing, you've been warned).

Both scripts depend on [ioport](https://people.redhat.com/~rjones/ioport/) to work, so make sure to install it first. Debian packages are available in main repository, and there is [a way](https://github.com/nekr0z/matebook-applet/issues/5) to get it on Arch, too.

* To get battery protection functionality:

    1. Download the [batpro script](https://github.com/nekr0z/linux-on-huawei-matebook-13-2019/blob/master/batpro), make it executable and copy to root executables path, i.e.:
        ```
        # mv batpro /usr/sbin/
        ```
    2. Allow your user to execute the script without providing sudo credentials. One way to do this would be to explicitly add sudoers permission (please be aware of possible security implications):
        ```
        # echo "your_username ALL = NOPASSWD: /usr/sbin/batpro" >> /etc/sudoers.d/batpro
        # chmod 0440 /etc/sudoers.d/batpro
        ```
  3. Double check that you can successfully run the script without providing additional authentication (i.e. password):
        ```
        $ sudo batpro status
        ```
* To get Fn-Lock functionality:

    1. Download the [fnlock script](https://github.com/nekr0z/linux-on-huawei-matebook-13-2019/blob/master/fnlock), make it executable and copy to root executables path, i.e.:
        ```
        # mv fnlock /usr/sbin/
        ```
    2. Allow your user to execute the script without providing sudo credentials. One way to do this would be to explicitly add sudoers permission (please be aware of possible security implications):

            # echo "your_username ALL = NOPASSWD: /usr/sbin/fnlock" >> /etc/sudoers.d/fnlock
            # chmod 0440 /etc/sudoers.d/fnlock

    3. Double check that you can successfully run the script without providing additional authentication (i.e. password):

            $ sudo fnlock status

</details>

### Compiling matebook-applet
You can always download precompiled amd64 binary from the [releases page](https://github.com/nekr0z/matebook-applet/releases), but it's also perfectly OK to compile matebook-applet yourself. Provided that you have the dependencies (GTK+ and libappindicator) installed (on Debian you can `sudo apt install libgtk-3-dev libappindicator3-dev`), all you need to do is:

        $ git clone https://github.com/nekr0z/matebook-applet.git
        $ cd matebook-applet
        $ go run build.go

## Usage
The user interface is intentionally as simple as they get. You get an icon in system tray that you can click and get a menu. The menu consists of current status, options to change it, and an option to quit the applet. Please be aware that the applet does not probe for current status on its own (this is intentional), so if you change your battery protection settings by other means it will not reflect the change. Clicking on the status line (top of the menu) updates it.

The entry that shows current Fn-Lock status is clickable, too, that toggles Fn-Lock (from ON to OFF or vice versa). Again, no probing here, so if you change Fn-Lock status by other means it will not reflect the change until clicked, but then it will toggle Fn-Lock again.

Command line option `-w` launches the applet in windowed (app) mode, i.e.:
```
$ matebook-applet -w
```

Other command line options can be found on the included manpage:
```
$ man -l matebook-applet.1
  or, if you installed applet from repository or .deb package,
$ man matebook-applet
```
### Gnome
As of Gnome 3.26 [the "legacy tray" is removed](https://bugzilla.gnome.org/show_bug.cgi?id=785956). Launching the applet in windowed (app) mode still works. An extension is needed for the system tray icon to show up, for example [AppIndicator](https://github.com/ubuntu/gnome-shell-extension-appindicator). 

## Development
Pull requests are always welcome!

### Contributing translations
> In case you don't want to install any Go tools, the notion of having to use `git` scares you, and you don't feel like figuring out what a pull request is and how to make one, but you still feel like contributing a translation, just drop me an email, we'll figure something out. ;-)

The intended way to go about contributing translations to the project is using [go-i18n](https://github.com/nicksnyder/go-i18n) tool. Make sure you have it installed and available, clone the repository, and:
```
$ cd assets/translations
$ ls
$ touch active.xx.toml
```
(where `xx` is your language)
```
$ goi18n merge active.*.toml
```
This will generate `translate.xx.toml` with all the lines that need to be translated to the language `xx`. Go ahead and translate it. Please make sure you delete all the messages you don't feel like translating before the next step, so that English messages don't creep in disguised as messages in your language.
```
$ goi18n merge active.*.toml translate.*.toml
```
Now all you've translated is in `active.xx.toml` and you can make a pull request.

## Credits
This software includes the following software or parts thereof:
* [getlantern/systray](https://github.com/getlantern/systray) Copyright © 2014 Brave New Software Project, Inc.
* [andlabs/ui](https://github.com/andlabs/ui) Copyright © 2014 Pietro Gagliardi
* [shurcooL/vfsgen](https://github.com/shurcooL/vfsgen) Copyright © 2015 Dmitri Shuralyov
* [BurntSushi/toml](https://github.com/BurntSushi/toml) Copyright © 2013 TOML authors
* [Jibber Jabber](https://github.com/cloudfoundry/jibber_jabber) Copyright © 2014 Pivotal
* [go-i18n](https://github.com/nicksnyder/go-i18n) Copyright © 2014 [Nick Snyder](https://github.com/nicksnyder)
* [The Go Programming Language](https://golang.org) Copyright © 2009 The Go Authors
* [Enkel Icons](https://www.deviantart.com/froyoshark/art/WIP-Enkel-Icon-Pack-for-Mac-498003378) by [FroyoShark](https://www.deviantart.com/froyoshark)

Translations:
* German - [Rouven "n3vu0r" Spreckles](https://github.com/n3vu0r)
* Russian - [Evgeny "nekr0z" Kuznetsov](https://evgenykuznetsov.org)

Packages are built using [fpm](https://github.com/jordansissel/fpm) and [nekr0z/changelog](https://github.com/nekr0z/changelog).

Big **THANK YOU** to [Ayman Bagabas](https://github.com/aymanbagabas) for all his work and support. Without him, there would be no matebook-applet.

Kudos to [Rouven Spreckels](https://github.com/n3vu0r) for sorting out `udev` and `systemd`.
