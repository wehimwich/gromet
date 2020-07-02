Gromet: the Go MET server
=========================

Gromet is a small server for multiplexing access to Paroscientific, Inc.
MET3/4/4A  Meteorological Measurement System and Handar TSI Company Ultrasonic
Wind Sensors. Currenlty only supports Perle serial-to-ethernet converter, though
local serial connections or other s2e devices may be added upon request.


Installation
------------

Install with

    git clone github.com/nvi-inc/gromet.git
    cd gromet
    make install

This installs gromet and configures it to run on startup.

Then edit the configuration in `/usr2/control/gromet.yml` point to your
serial-to-ethernet converter, and start gromet with

    systemctl --user start gromet

Note, this installation assumes you are using standard FS Linux directoires (under `/usr2`)
for configuration and binaries and that you are using a systemd
based OS. If this do not match your setup, edit the makefile appropriately.

Upgrading
---------

To upgrade, fetch the new source and reinstall

    cd path/to/gromet
    git pull
    make install

You will be prompted to overwrite your configuration or not.
