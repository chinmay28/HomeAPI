package io.github.chinmay28.homeapi

import android.app.PendingIntent
import android.appwidget.AppWidgetManager
import android.appwidget.AppWidgetProvider
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.view.View
import android.widget.RemoteViews
import io.github.chinmay28.homeapi.core.EntryWidget
import io.github.chinmay28.homeapi.core.WidgetContent
import java.net.HttpURLConnection
import java.net.URL
import java.text.DateFormat
import java.util.Date
import java.util.concurrent.Executors

/**
 * The home-screen widget: one entry's value, pinned by key — a watchlist, a
 * note, a config value a script keeps current — read from whichever server
 * the app is set to. Tapping it reads it again; Android also refreshes it
 * every half hour (the shortest period a widget may ask for), and the app
 * refreshes every widget when the server on the phone comes up.
 *
 * What it asks and how the answer reads is [EntryWidget]; this is the glue.
 */
class EntryWidgetProvider : AppWidgetProvider() {
    override fun onUpdate(context: Context, manager: AppWidgetManager, ids: IntArray) {
        refresh(context, ids)
    }

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == ACTION_REFRESH) {
            val id = intent.getIntExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, AppWidgetManager.INVALID_APPWIDGET_ID)
            if (id != AppWidgetManager.INVALID_APPWIDGET_ID) refresh(context, intArrayOf(id))
            return
        }
        super.onReceive(context, intent)
    }

    override fun onDeleted(context: Context, ids: IntArray) {
        val edit = prefs(context).edit()
        for (id in ids) edit.remove(keyPref(id))
        edit.apply()
    }

    private fun refresh(context: Context, ids: IntArray) {
        val pending = goAsync()
        val app = context.applicationContext
        pool.execute {
            try {
                val manager = AppWidgetManager.getInstance(app)
                for (id in ids) {
                    val key = prefs(app).getString(keyPref(id), null)
                    manager.updateAppWidget(id, views(app, id, key, loading = true, content = null))
                    val content = if (key == null) WidgetContent.NotSet else fetch(app, key)
                    manager.updateAppWidget(id, views(app, id, key, loading = false, content = content))
                }
            } finally {
                pending.finish()
            }
        }
    }

    companion object {
        const val ACTION_REFRESH = "io.github.chinmay28.homeapi.REFRESH_WIDGET"
        private const val PREFS = "widgets"
        private val pool = Executors.newSingleThreadExecutor()

        fun keyPref(id: Int) = "key_$id"
        fun prefs(context: Context) = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

        /** Reads every placed widget again. */
        fun refreshAll(context: Context) {
            val manager = AppWidgetManager.getInstance(context)
            val ids = manager.getAppWidgetIds(ComponentName(context, EntryWidgetProvider::class.java))
            if (ids.isEmpty()) return
            context.sendBroadcast(
                Intent(context, EntryWidgetProvider::class.java)
                    .setAction(AppWidgetManager.ACTION_APPWIDGET_UPDATE)
                    .putExtra(AppWidgetManager.EXTRA_APPWIDGET_IDS, ids),
            )
        }

        private fun fetch(context: Context, key: String): WidgetContent {
            val origin = ConfigStore(context).load().origin().getOrElse {
                return WidgetContent.Problem(key, context.getString(R.string.widget_not_configured))
            }
            return try {
                val conn = URL(origin.url(EntryWidget.path(key))).openConnection() as HttpURLConnection
                conn.connectTimeout = 5_000
                conn.readTimeout = 10_000
                conn.instanceFollowRedirects = false
                try {
                    val code = conn.responseCode
                    val body = (if (code < 400) conn.inputStream else conn.errorStream)?.bufferedReader()?.use { it.readText() } ?: ""
                    EntryWidget.read(key, code, body)
                } finally {
                    conn.disconnect()
                }
            } catch (_: Exception) {
                WidgetContent.Problem(key, context.getString(R.string.widget_unreachable, origin.toString()))
            }
        }

        private fun views(context: Context, id: Int, key: String?, loading: Boolean, content: WidgetContent?): RemoteViews {
            val v = RemoteViews(context.packageName, R.layout.widget_entry)
            val tap = if (key == null) {
                // Not set up (the configure screen was skipped): tapping sets it up.
                Intent(context, EntryWidgetConfigActivity::class.java)
                    .putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, id)
                    .let { PendingIntent.getActivity(context, id, it, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT) }
            } else {
                Intent(context, EntryWidgetProvider::class.java)
                    .setAction(ACTION_REFRESH)
                    .putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, id)
                    .let { PendingIntent.getBroadcast(context, id, it, PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT) }
            }
            v.setOnClickPendingIntent(R.id.widget_root, tap)
            v.setViewVisibility(R.id.widget_progress, if (loading) View.VISIBLE else View.GONE)
            if (loading && content == null) {
                v.setTextViewText(R.id.widget_key, key ?: context.getString(R.string.app_name))
                return v
            }
            val (title, body, footer) = when (content) {
                is WidgetContent.Value -> Triple(content.key, content.text.ifEmpty { context.getString(R.string.widget_empty) }, updated(context))
                is WidgetContent.Missing -> Triple(content.key, context.getString(R.string.widget_missing), updated(context))
                is WidgetContent.Problem -> Triple(content.key, content.message, context.getString(R.string.widget_tap_retry))
                else -> Triple(context.getString(R.string.app_name), context.getString(R.string.widget_not_set), "")
            }
            v.setTextViewText(R.id.widget_key, title)
            v.setTextViewText(R.id.widget_value, body)
            v.setTextViewText(R.id.widget_footer, footer)
            return v
        }

        private fun updated(context: Context): String =
            context.getString(R.string.widget_updated, DateFormat.getTimeInstance(DateFormat.SHORT).format(Date()))
    }
}
