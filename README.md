# matebook-applet
System tray applet/control app for Huawei Matebook

This simple applet is designed to make some of the proptietary Huawei PC Manager's functionality available via GUI on Linux.
 
![Applet screenshot](screenshot-applet.png?raw=true "matebook-applet")

It can be used as a system tray applet (above) or as a windowed app (below).

![Windowed screenshot](screenshot-windowed.png?raw=true "matebook-applet windowed mode")

##### Table of Contents
* [Installation and setup](#installation-and-setup)
  * [Huawei-WMI driver](#huawei-wmi-driver) - for those running Linux kernel 5.0 or later
  * [EC scripts](#embedded-controller-scripts) - for those with older kernel
  * [Compiling](#compiling-matebook-applet)
* [Usage](#usage)
* [Development](#development)
* [Credits](#credits)

## Installation and setup
The applet requires no installation as such (although you may want to add it to autorun so that it gets started automatically when you load your DE). However, it depends on GTK+ and `libappindicator` to be installed on your system, and it relies on underlying pieces of software to be installed so that it is user-accessible (more on that below).

Debian users can add this repo:
```
deb http://evgenykuznetsov.org/repo/ buster main
```
to their `/etc/apt/sources.list` and add my public key:
```
$ sudo apt-key adv --keyserver pool.sks-keyservers.net --recv 79B2B092A63F5E6F
```
From there, the applet is just a `sudo apt install matebook-applet` away.

### Huawei-WMI driver
Starting with version 1.2 the best way to get the matebook-applet working is to install [Huawei-WMI driver](https://github.com/aymanbagabas/Huawei-WMI) by [Ayman Bagabas](https://github.com/aymanbagabas). You'll need at least version 3.0 of his driver. Be advised that the driver is only compatible with Linux kernel 5.0 or later.

After installing the driver you may start using the matebook-applet right away. However, it will only be able to display the current settings, not change them. In order to do that you either need to run the applet as root (absolutely not recommended), or make sure all the necessary files (the hooks in `/sys/devices/platform/huawei-wmi` as well as `/etc/default/huawei-wmi/` directory) are user-writable. A good way to set everything up is to make use of [Rouven Spreckels](https://github.com/n3vu0r)' awesome [project](https://github.com/qu1x/huawei-wmi). Debian users will have it installed from the abovementioned repository, the rest can:
```
$ git clone https://github.com/qu1x/huawei-wmi.git
$ cd huawei-wmi/generic
$ sudo make install
```
You may need to re-login for adding your user to group to take effect.

### Embedded controller scripts

On Linux kernels earlier than 5.0 you can make this matebook-applet work using the two scripts as explained below. Any one of them is enough, but you need both for full functionality. You may download them both in one archive [here](https://github.com/nekr0z/linux-on-huawei-matebook-13-2019/releases). Please note that these scripts are for MateBook 13 and will not work on other MateBooks (you can still get them to work by changing them to address EC registers proper for your particular model, but make sure you know exactly what you're doing, you've been warned).

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

### Compiling matebook-applet
You can always download precompiled amd64 binary from [releases page](https://github.com/nekr0z/matebook-applet/releases), but it's also perfectly OK to compile matebook-applet yourself. Provided that you have the dependencies (GTK+ and libappindicator) installed (on Debian you can `sudo apt install libgtk-3-dev libappindicator3-dev`), all you need to do is:

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

## Development
PRs are most welcome!

## Credits
This software includes the following software or parts thereof:
* [getlantern/systray](https://github.com/getlantern/systray)
* [andlabs/ui](https://github.com/andlabs/ui) Copyright © 2014 Pietro Gagliardi
* [slurcooL/vfsgen](https://github.com/shurcooL/vfsgen) Copyright © 2015 Dmitri Shuralyov
* [The Go Programming Language](https://golang.org) Copyright © 2009 The Go Authors
* [Enkel Icons](https://www.deviantart.com/froyoshark/art/WIP-Enkel-Icon-Pack-for-Mac-498003378) by [FroyoShark](https://www.deviantart.com/froyoshark)

Packages are built using [fpm](https://github.com/jordansissel/fpm) and [nekr0z/changelog](https://github.com/nekr0z/changelog).

Big **THANK YOU** to [Ayman Bagabas](https://github.com/aymanbagabas) for all his work and support. Without him, there would be no matebook-applet.

Kudos to [Rouven Spreckels](https://github.com/n3vu0r) for sorting out `udev` and `systemd`.
