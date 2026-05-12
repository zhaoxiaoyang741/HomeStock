#!/bin/bash
APP="./server-linux-amd64"
LOG="./logger.log"
PIDFILE="./server.pid"

start() {
    if [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
        echo "server is already running (pid $(cat "$PIDFILE"))"
        exit 1
    fi
    nohup "$APP" > "$LOG" 2>&1 &
    echo $! > "$PIDFILE"
    echo "server started (pid $(cat "$PIDFILE"))"
}

stop() {
    if [ ! -f "$PIDFILE" ]; then
        echo "pid file not found"
        exit 1
    fi
    pid=$(cat "$PIDFILE")
    if ! kill -0 "$pid" 2>/dev/null; then
        echo "server not running (stale pid $pid)"
        rm -f "$PIDFILE"
        exit 1
    fi
    kill "$pid"
    # Wait up to 10s for graceful shutdown
    for i in $(seq 1 10); do
        if ! kill -0 "$pid" 2>/dev/null; then
            break
        fi
        sleep 1
    done
    if kill -0 "$pid" 2>/dev/null; then
        echo "server did not stop gracefully, force killing..."
        kill -9 "$pid"
    fi
    rm -f "$PIDFILE"
    echo "server stopped"
}

case "${1:-}" in
    start)   start ;;
    stop)    stop ;;
    restart) stop; sleep 1; start ;;
    *)       echo "usage: $0 {start|stop|restart}" ;;
esac
