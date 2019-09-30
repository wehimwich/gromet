Gromet: the Go MET server
=========================

Gromet is a small server for multiplexing access
to Paroscientific, Inc. MET3/4/4A  Meteorological Measurement System
and Handar TSI Company Ultrasonic Wind Sensors.


Installation
------------


Install with 

    git clone github.com/nvi-inc/gromet.git
    make

Setup by running

     mkdir -p $HOME/.config/systemd/user/
     cp gromet.service $HOME/.config/systemd/user/
     cp gromet.yml /usr2/control # (or /etc/gromet if more appropriate )
     systemctl --user enable gromet
