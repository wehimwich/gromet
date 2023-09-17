_gromet_: the _go_ MET server
=========================

_gromet_ is a small server for multiplexing access to Paroscientific,
Inc. MET3/4/4A  Meteorological Measurement System and Vaisala WMT70x
Ultrasonic Wind Sensors. It currenlty only supports Perle
serial-to-ethernet converter, though local serial connections or other
s2e devices may be added upon request.

Installation
------------

(You must have the _go_ language installed. With FSLx this is usually done
with _fsadapt_ as _root_.)

As _root_:

    cd /usr2
    git clone https://github.com/nvi-inc/gromet.git
    chmod -R prog.rtx gromet

As _prog_:

    cat <<EOF >>~/.profile
    export GOPATH=~/go
    PATH="$GOPATH/bin:/usr/local/go/bin:$PATH"
    EOF
    . ~/.profile
    cd /usr2/gromet
    make

As _root_:

    git config --global --add safe.directory /usr2/gromet
    make install

This installs _gromet_ and configures it to run on startup.

Then edit the configuration in _/usr2/control/gromet.yml_ point to
your serial-to-ethernet converter, and start _gromet_ with

    systemctl start gromet

Note, this installation assumes you are using standard FS Linux
directories (under _/usr2_) and user _oper_ and that you are using a
`systemd` based OS. If this do not match your setup, edit the
_Makefile_ appropriately.

Upgrading
---------

To upgrade, fetch the new source and reinstall

    cd /usr2/gromet
    git pull
    make

If an update to the service is needed, then as _root_:

    make install
    systemctl restart gromet

You will be prompted to overwrite your configuration or not.
Typically, you don't want to overwrite _/usr2/control/gromet.yml_ but
it may need to be updated.
