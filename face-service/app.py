"""Entry point — starts Flask service + monitor thread."""

import os

from monitor import start_monitor_thread
from service import app

if __name__ == "__main__":
    # Start monitor loop in background
    monitor = start_monitor_thread()

    port = int(os.environ.get("PORT", 5050))
    app.run(host="0.0.0.0", port=port, debug=False)
