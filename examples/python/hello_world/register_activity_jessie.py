from cakework.app import App
from activity import activity

app = App("jessiesapp")
app.register_activity(activity, "activity")