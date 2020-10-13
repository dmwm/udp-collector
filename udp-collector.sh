#/bin/bash
##H Usage: udp-collector.sh <start|stop|status|restart>
##H
##H   status      show current service's status
##H   sysboot     start server from crond if not running
##H   restart     (re)start the service
##H   start       (re)start the service
##H   stop        stop the service
##H

wdir=$PWD
if [ "`hostname -s`" == "vomcs0200" ]; then
    wdir=/opt/udp
fi
echo "Work dir $wdir"

# start actions
start_udp_server()
{
    # start udp_server
    nohup $wdir/udp_server -config $wdir/udp_server.json 2>&1 1>& /dev/null < /dev/null &
    pid=`ps auxwww | egrep "udp_server -config" | egrep -v "grep|process_monitor" | awk 'BEGIN{ORS=" "} {print $2}'`
    echo "Started udp_server service... PID=${pid}"

    # start udp_server_monitor which monitor/maintain udp_server running status
    nohup $wdir/udp_server_monitor -config $wdir/udp_server.json 2>&1 1>& $wdir/udp_server.log < /dev/null &
    pid=`ps auxwww | egrep "udp_server_monitor -config" | egrep -v "grep|process_monitor" | awk 'BEGIN{ORS=" "} {print $2}'`
    echo "Started udp_server_monitor service... PID=${pid}"
}
start_node_exporter()
{
    if [ -f $wdir/node_exporter ]; then
        # start node_exporter to get metrics about our node
        nohup $wdir/node_exporter 2>&1 1>& $wdir/node_exporter.log < /dev/null &
        pid=`ps auxwww | egrep "node_exporter" | egrep -v "grep" | awk 'BEGIN{ORS=" "} {print $2}'`
        echo "Started node_exporter service... PID=${pid}"
    else
        echo "No $wdir/node_exporter found ..."
    fi
}
start_proc_exporter()
{
    if [ -f $wdir/process_monitor.sh ] && [ -f $wdir/process_exporter ]; then
        # start process_monitor which starts process_exporter to collect metrics about udp_server
        nohup $wdir/process_monitor.sh "udp_server -config udp_server.json" udp_server ":9101" 5 2>&1 1>& $wdir/process_monitor.log < /dev/null &
        pid=`ps auxwww | egrep "process_monitor.sh" | egrep -v "grep" | awk 'BEGIN{ORS=" "} {print $2}'`
        echo "Started process_monitor.sh service... PID=${pid}"
    else
        echo "No $wdir/process_monitor.sh and $wdir/process_exporter found ..."
    fi
}

# stop actions
stop_udp_server()
{
    local pid=`ps auxwww | egrep "udp_server" | egrep -v "grep|process_monitor" | awk 'BEGIN{ORS=" "} {print $2}'`
    echo "Stop udp_server service... PID=${pid}"
    if [ -n "${pid}" ]; then
        kill -9 ${pid}
    fi
}
stop_node_exporter()
{
    local pid=`ps auxwww | egrep "node_exporter" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}'`
    echo "Stop node_exporter service... PID=${pid}"
    if [ -n "${pid}" ]; then
        kill -9 ${pid}
    fi
}
stop_proc_exporter()
{
    local pid=`ps auxwww | egrep "process_exporter" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}'`
    echo "Stop process_exporter service... PID=${pid}"
    if [ -n "${pid}" ]; then
        kill -9 ${pid}
    fi
}
# status actions
status_udp_server()
{
    local pid=`ps auxwww | egrep "udp_server" | egrep -v "grep|process_monitor" | awk 'BEGIN{ORS=" "} {print $2}'`
    if  [ -z "${pid}" ]; then
        echo "udp_server is not running"
    else
        echo "udp_server is running: PID=$pid"
    fi
}
status_node_exporter()
{
    local pid=`ps auxwww | egrep "node_exporter" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}'`
    if  [ -z "${pid}" ]; then
        echo "node_exporter is not running"
    else
        echo "node_exporter is running: PID=$pid"
    fi
}
status_proc_exporter()
{
    local pid=`ps auxwww | egrep "process_exporter" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}'`
    if  [ -z "${pid}" ]; then
        echo "process_exporter is not running"
    else
        echo "process_exporter is running: PID=$pid"
    fi
}

case "$1" in
start | restart)
   stop_udp_server
   stop_node_exporter
   stop_proc_exporter
   start_udp_server
   start_node_exporter
   start_proc_exporter
   ;;
stop)
   stop_udp_server
   stop_node_exporter
   stop_proc_exporter
   ;;
status)
   status_udp_server
   status_node_exporter
   status_proc_exporter
   ;;
*)
   echo "Usage: $0 {start|stop|status|restart}"
esac

exit 0
