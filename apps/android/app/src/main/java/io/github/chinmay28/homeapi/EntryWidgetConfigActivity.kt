package io.github.chinmay28.homeapi

import android.appwidget.AppWidgetManager
import android.content.Intent
import android.os.Bundle
import androidx.activity.enableEdgeToEdge
import androidx.appcompat.app.AppCompatActivity
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import com.google.android.material.button.MaterialButton
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import io.github.chinmay28.homeapi.core.EntryWidget

/**
 * Choosing the entry a widget shows: shown when the widget is placed, and
 * again from the widget's own settings on a launcher that offers them.
 */
class EntryWidgetConfigActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        val id = intent?.getIntExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, AppWidgetManager.INVALID_APPWIDGET_ID)
            ?: AppWidgetManager.INVALID_APPWIDGET_ID
        // Backing out of a placement leaves no widget behind.
        setResult(RESULT_CANCELED, Intent().putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, id))
        if (id == AppWidgetManager.INVALID_APPWIDGET_ID) {
            finish()
            return
        }

        setContentView(R.layout.activity_widget_config)
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.root)) { v, insets ->
            val bars = insets.getInsets(WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.ime())
            v.setPadding(bars.left, bars.top, bars.right, bars.bottom)
            WindowInsetsCompat.CONSUMED
        }
        val layout = findViewById<TextInputLayout>(R.id.key_layout)
        val key = findViewById<TextInputEditText>(R.id.key)
        val prefs = EntryWidgetProvider.prefs(this)
        key.setText(prefs.getString(EntryWidgetProvider.keyPref(id), ""))

        findViewById<MaterialButton>(R.id.save).setOnClickListener {
            val k = EntryWidget.key(key.text.toString()).getOrElse {
                layout.error = it.message
                return@setOnClickListener
            }
            prefs.edit().putString(EntryWidgetProvider.keyPref(id), k).apply()
            sendBroadcast(
                Intent(this, EntryWidgetProvider::class.java)
                    .setAction(AppWidgetManager.ACTION_APPWIDGET_UPDATE)
                    .putExtra(AppWidgetManager.EXTRA_APPWIDGET_IDS, intArrayOf(id)),
            )
            setResult(RESULT_OK, Intent().putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, id))
            finish()
        }
    }
}
