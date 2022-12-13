from sahale.app import App
from submodule.activity import activity

app = App("id", "user")
app.register_activity(activity, "activity")