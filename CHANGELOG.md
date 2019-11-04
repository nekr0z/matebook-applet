# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- internationalization

## [2.3.3] - 2019-10-31
### Fixed
- build script

## [2.3.2] - 2019-10-31
### Fixed
- release script

## [2.3.1] - 2019-10-31
### Fixed
- lack of proper changelog
- displayed version numbers

## [2.3.0] - 2019-09-22
### Added
- windowed mode (triggered by `-w` option)
- deprecation warning when `-s` option is used
### Fixed
- crashes in cases where either only fnlock or only battery endpoint is available
- lack of documentation about GTK+ dependency

## [2.2.0] - 2019-09-21
### Added
- a way to set custom battery threshold

## [2.1.4] - 2019-09-19
### Changed
- platform endpoints are now preferred instead of kernel ones
### Fixed
- persistent battery thresholds endpoint autodetection
- minor glitches in `*.deb` generation

## [2.1.3] - 2019-09-16
### Added
- `-r` option for using scripts as endpoints; on by default for now, but will be off by default eventually
### Fixed
- `batpro` script not working
- minor glitches

## [2.1.2] - 2019-09-15
### Fixed
- support for some corner cases and legacy endpoints

## [2.1.1] - 2019-09-11
### Fixed
- minor issues

## [2.1.0] - 2019-09-09
### Added
- support for driver version >3.2

## [2.0.2] - 2019-09-09
### Fixed
- battery status message when status is unavailable

## [2.0.1] - 2019-06-18
### Fixed
- minor manpage defects

## [2.0.0] - 2019-06-17
### Added
- `.desktop` file
- `-n` option for *not* saving battery thresholds for persistence
### Removed
- attempts to `chmod` files in `/etc/default/huawei-wmi/`
### Deprecated
- `-s` option ("save battery thresholds for persistence")

## [1.3.3] - 2019-06-10
### Fixed
- lack of manpage
- `.deb` did not depend on `libc6`

## [1.3.2] - 2019-06-07
### Fixed
- lack of `.deb` in release

## [1.3.1] - 2019-06-06
### Removed
- obsolete files in release

## [1.3.0] - 2019-06-06
### Added
- `-s` option for saving battery thresholds for persistence
### Changed
- outsorced udev rules to [qu1x/huawei-wmi](https://github.com/qu1x/huawei-wmi) project
### Fixed
- build process

## [1.2.5] - 2019-05-02
### Added
- `-wait` option for MateBook X
### Removed
- driver waits by default

## [1.2.4] - 2019-04-28
### Fixed
- battery protection status messages inconsistency

## [1.2.3] - 2019-04-28
### Added
- `-icon` option for custom icon
### Fixed
- didn't wait for driver to set thresholds
- minor glitches

## [1.2.2] - 2019-04-26
### Fixed
- icon was invisible on dark panel background

## [1.2.1] - 2019-04-26
### Fixed
- no version info was shown
- minor glitches

## [1.2.0] - 2019-04-25
### Added
- support for Huawei-WMI driver

## [1.1.0] - 2019-04-19
### Added
- Fn-Lock functionality
### Fixed
- minor glitches

## [1.0.1] - 2019-04-19
### Fixed
- icon not showing up

## 1.0.0 - 2019-04-19
*initial release*
### Added
- setting battery protection thresholds
- checks for scripts availability

[Unreleased]: https://github.com/nekr0z/matebook-applet/compare/2.3.3...HEAD
[2.3.3]: https://github.com/nekr0z/matebook-applet/compare/2.3.2...2.3.3
[2.3.2]: https://github.com/nekr0z/matebook-applet/compare/2.3.1...2.3.2
[2.3.1]: https://github.com/nekr0z/matebook-applet/compare/v2.3.0...2.3.1
[2.3.0]: https://github.com/nekr0z/matebook-applet/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/nekr0z/matebook-applet/compare/v2.1.4...v2.2.0
[2.1.4]: https://github.com/nekr0z/matebook-applet/compare/v2.1.3...v2.1.4
[2.1.3]: https://github.com/nekr0z/matebook-applet/compare/v2.1.2...v2.1.3
[2.1.2]: https://github.com/nekr0z/matebook-applet/compare/v2.1.1...v2.1.2
[2.1.1]: https://github.com/nekr0z/matebook-applet/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/nekr0z/matebook-applet/compare/v2.0.2...v2.1.0
[2.0.2]: https://github.com/nekr0z/matebook-applet/compare/v2.0.1...v2.0.2
[2.0.1]: https://github.com/nekr0z/matebook-applet/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/nekr0z/matebook-applet/compare/v1.3.3...v2.0.0
[1.3.3]: https://github.com/nekr0z/matebook-applet/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/nekr0z/matebook-applet/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/nekr0z/matebook-applet/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/nekr0z/matebook-applet/compare/v1.2.5...v1.3.0
[1.2.5]: https://github.com/nekr0z/matebook-applet/compare/v1.2.4...v1.2.5
[1.2.4]: https://github.com/nekr0z/matebook-applet/compare/v1.2.3...v1.2.4
[1.2.3]: https://github.com/nekr0z/matebook-applet/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/nekr0z/matebook-applet/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/nekr0z/matebook-applet/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/nekr0z/matebook-applet/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/nekr0z/matebook-applet/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/nekr0z/matebook-applet/compare/v1.0.0...v1.0.1
