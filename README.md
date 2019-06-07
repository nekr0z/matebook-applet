# matebook-applet
System tray applet for Huawei Matebook

This simple applet is designed to make some of the proptietary Huawei PC Manager's functionality available via GUI on Linux.

##### Table of Contents
* [Installation and setup](#installation-and-setup)
  * [Huawei-WMI driver](#huawei-wmi-driver) - for those running Linux kernel 5.0 or later
  * [EC scripts](#embedded-controller-scripts) - for those with older kernel
  * [Compiling](#compiling-matebook-applet)
* [Usage](#usage)
* [Development](#development)
* [Credits](#credits)

## Installation and setup
The applet requires no installation as such (although you may want to add it to autorun so that it gets started automatically when you load your DE). However, it depends on `libappindicator` to be installed on your system (`sudo apt install libappindicator3-1` for Debian), and it relies on underlying pieces of software to be installed so that it is user-accessible:

### Huawei-WMI driver
Starting with version 1.2 the best way to get the matebook-applet working is to install [Huawei-WMI driver](https://github.com/aymanbagabas/Huawei-WMI) by [Ayman Bagabas](https://github.com/aymanbagabas). You'll need at least version 3.0 of his driver. Be advised that the driver is only compatible with Linux kernel 5.0 or later.

After installing the driver you may start using the matebook-applet right away. However, it will only be able to display the current settings, not change them. In order to do that you either need to run the applet as root (absolutely not recommended), or make sure all the necessary files (the hooks in `/sys/devices/platform/huawei-wmi` as well as `/etc/default/huawei-wmi/` directory) are user-writable. A good way to set everything up is to make use of [Rouven Spreckels](https://github.com/n3vu0r)' awesome [project](https://github.com/qu1x/huawei-wmi):
```
$ git clone https://github.com/qu1x/huawei-wmi.git

$ cd huawei-wmi/generic
 or
$ cd huawei-wmi/debian

$ sudo make install
```
You may need to re-login for adding your user to group to take effect.

Debian users may download `.deb` package from [releases page](https://github.com/nekr0z/matebook-applet/releases).

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
You can always download precompiled amd64 binary from [releases page](https://github.com/nekr0z/matebook-applet/releases), but it's also perfectly OK to compile matebook-applet yourself:

        $ git clone https://github.com/nekr0z/matebook-applet.git
        $ cd matebook-applet
        $ go run build.go

## Usage
The user interface is intentionally as simple as they get. You get an icon in system tray that you can click and get a menu. The menu consists of current status, options to change it, and an option to quit the applet. Please be aware that the applet does not probe for current status on its own (this is intentional), so if you change your battery protection settings by other means it will not reflect the change. Clicking on the status line (top of the menu) updates it.

The entry that shows current Fn-Lock status is clickable, too, that toggles Fn-Lock (from ON to OFF or vice versa). Again, no probing here, so if you change Fn-Lock status by other means it will not reflect the change until clicked, but then it will toggle Fn-Lock again.

Command line options include `-v` that gives more information in `stdout` while running the applet. `-vv` gives even more information, and is useful for debugging if you are having an issue and need to ask for help. You can also specify a custom icon to show in system tray using `-icon /path/to/your/icon` (SVG, PNG or ICO should work).

`-s` command line switch allows for saving battery threshold settings to `/etc/default/huawei-wmi/` so that they can later be restored by systemd unit after reboots and prolonged powerdowns. This option only works with Huawei-WMI driver.

## Development
PRs are most welcome!

## Credits
This software includes the following software or parts thereof:
* [getlantern/systray](https://github.com/getlantern/systray)
* [slurcooL/vfsgen](https://github.com/shurcooL/vfsgen) Copyright © 2015 Dmitri Shuralyov
* [The Go Programming Language](https://golang.org) Copyright © 2009 The Go Authors
* [Enkel Icons](https://www.deviantart.com/froyoshark/art/WIP-Enkel-Icon-Pack-for-Mac-498003378) by [FroyoShark](https://www.deviantart.com/froyoshark)

Big **THANK YOU** to [Ayman Bagabas](https://github.com/aymanbagabas) for all his work and support. Without him, there would be no matebook-applet.

Kudos to [Rouven Spreckels](https://github.com/n3vu0r) for sorting out `udev` and `systemd`.
