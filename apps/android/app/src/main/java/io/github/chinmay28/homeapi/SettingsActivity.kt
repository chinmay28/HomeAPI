package io.github.chinmay28.homeapi

import android.content.Intent
import android.os.Bundle
import android.text.InputType
import android.view.View
import android.widget.TextView
import androidx.activity.enableEdgeToEdge
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.preference.EditTextPreference
import androidx.preference.Preference
import androidx.preference.PreferenceCategory
import androidx.preference.PreferenceFragmentCompat
import androidx.preference.SwitchPreferenceCompat
import com.google.android.material.appbar.MaterialToolbar
import io.github.chinmay28.homeapi.core.AppMode
import io.github.chinmay28.homeapi.core.ModeChange
import io.github.chinmay28.homeapi.core.ModeSwitch
import io.github.chinmay28.homeapi.core.Origin
import io.github.chinmay28.homeapi.core.ServerState
import kotlin.concurrent.thread

/**
 * The app's own settings — which server, and how the one on the phone runs.
 * Everything about the data itself is in HomeAPI's web GUI.
 *
 * Changes are saved as they are made; the main screen picks them up when it
 * comes back, restarting the server or reconnecting as needed.
 */
class SettingsActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)
        setSupportActionBar(findViewById<MaterialToolbar>(R.id.toolbar))
        supportActionBar?.setDisplayHomeAsUpEnabled(true)

        // Edge to edge (always, from Android 15): the toolbar's background
        // runs up under the status bar while its contents stay below it, and
        // the list keeps clear of the navigation bar and any cutout.
        val appBar = findViewById<View>(R.id.app_bar)
        val container = findViewById<View>(R.id.settings_container)
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.root)) { _, insets ->
            val bars = insets.getInsets(WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.displayCutout() or WindowInsetsCompat.Type.ime())
            appBar.setPadding(bars.left, bars.top, bars.right, 0)
            container.setPadding(bars.left, 0, bars.right, bars.bottom)
            WindowInsetsCompat.CONSUMED
        }

        if (savedInstanceState == null) {
            supportFragmentManager.beginTransaction().replace(R.id.settings_container, SettingsFragment()).commit()
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    class SettingsFragment : PreferenceFragmentCompat() {
        private lateinit var store: ConfigStore
        private var unobserve: (() -> Unit)? = null

        override fun onCreatePreferences(savedInstanceState: Bundle?, rootKey: String?) {
            preferenceManager.sharedPreferencesName = ConfigStore.FILE
            setPreferencesFromResource(R.xml.preferences, rootKey)
            store = ConfigStore(requireContext())
            val canRunHere = LocalServer.available(requireContext())

            // The mode itself lives under KEY_MODE, as ConfigStore reads it;
            // the switch only shows it and writes it back.
            val runOnPhone = findPreference<SwitchPreferenceCompat>(KEY_RUN_ON_PHONE)!!
            runOnPhone.isChecked = store.load().mode == AppMode.ON_PHONE
            runOnPhone.setOnPreferenceChangeListener { _, on ->
                val target = if (on == true) AppMode.ON_PHONE else AppMode.REMOTE
                when (ModeSwitch.check(target, store.load(), canRunHere)) {
                    ModeChange.Unavailable -> {
                        message(getString(R.string.setup_phone_unavailable))
                        false
                    }
                    ModeChange.Apply -> {
                        setMode(target)
                        true
                    }
                    ModeChange.NeedsAddress -> {
                        setMode(target)
                        // A thin client with nowhere to go: ask where, now.
                        listView.post { onDisplayPreferenceDialog(findPreference<EditTextPreference>(ConfigStore.KEY_ADDRESS)!!) }
                        true
                    }
                }
            }

            findPreference<EditTextPreference>(ConfigStore.KEY_ADDRESS)!!.apply {
                setOnBindEditTextListener { it.inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_URI }
                setOnPreferenceChangeListener { pref, v ->
                    Origin.parse(v as String).fold(
                        onSuccess = { o ->
                            // Stored the way it will be read back: the origin, spelled out.
                            (pref as EditTextPreference).text = o.toString()
                            false
                        },
                        onFailure = { message(it.message ?: "That is not an address."); false },
                    )
                }
            }

            findPreference<Preference>("find_servers")!!.setOnPreferenceClickListener { discover(); true }

            findPreference<EditTextPreference>(ConfigStore.KEY_PORT)!!.apply {
                if (text == null) text = Origin.DEFAULT_PORT.toString()
                setOnBindEditTextListener { it.inputType = InputType.TYPE_CLASS_NUMBER }
                setOnPreferenceChangeListener { _, v ->
                    val ok = (v as String).toIntOrNull() in 1024..65535
                    if (!ok) message(getString(R.string.pref_port) + ": 1024–65535")
                    ok
                }
            }

            findPreference<EditTextPreference>(ConfigStore.KEY_NAME)!!.apply {
                setOnBindEditTextListener { it.hint = store.defaultDeviceName() }
                summaryProvider = Preference.SummaryProvider<EditTextPreference> {
                    getString(R.string.pref_name_summary, store.deviceName())
                }
            }
            findPreference<SwitchPreferenceCompat>(ConfigStore.KEY_SHARE)!!.setOnPreferenceChangeListener { _, on ->
                if (on == true) {
                    ServerService.lanAddress()?.let { ip ->
                        message(getString(R.string.notify_server_shared, ip, store.load().localPort) + "\n\n" + getString(R.string.pref_share_summary_on))
                    }
                }
                true
            }

            findPreference<Preference>("server_log")!!.setOnPreferenceClickListener { showLog(); true }
            findPreference<Preference>("restart_server")!!.setOnPreferenceClickListener {
                LocalServer.restart(requireContext()); true
            }
            findPreference<Preference>("stop_server")!!.setOnPreferenceClickListener {
                LocalServer.stop(requireContext()); true
            }
            findPreference<Preference>("version")!!.summary = BuildConfig.VERSION_NAME +
                if (canRunHere) "" else " · " + getString(R.string.status_unavailable)
            findPreference<Preference>("developer")!!.setOnPreferenceClickListener {
                DevBadge.flash(requireActivity()); true
            }

            showFor(store.load().mode)
        }

        override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
            super.onViewCreated(view, savedInstanceState)
            // The last rows scroll up clear of the navigation bar, rather than
            // the list stopping short of it with a dead band underneath.
            listView.clipToPadding = false
        }

        private fun setMode(m: AppMode) {
            preferenceManager.sharedPreferences!!.edit().putString(ConfigStore.KEY_MODE, ConfigStore.modeValue(m)).apply()
            showFor(m)
        }

        override fun onStart() {
            super.onStart()
            val status = findPreference<Preference>("server_status")!!
            unobserve = LocalServer.observe { s ->
                activity?.runOnUiThread { status.summary = describe(s) }
            }
        }

        override fun onStop() {
            unobserve?.invoke()
            unobserve = null
            super.onStop()
        }

        private fun describe(s: ServerState): String = when {
            !LocalServer.available(requireContext()) -> getString(R.string.status_unavailable)
            s is ServerState.Running -> getString(R.string.status_running, s.version, Origin.loopback(store.load().localPort).toString())
            s is ServerState.Starting -> getString(R.string.status_starting)
            s is ServerState.Restarting -> getString(R.string.status_restarting, s.exitCode)
            s is ServerState.Failed -> getString(R.string.status_failed, s.exitCode)
            else -> getString(R.string.status_stopped)
        }

        private fun showFor(mode: AppMode?) {
            val phone = mode == AppMode.ON_PHONE
            findPreference<PreferenceCategory>("cat_remote")!!.isVisible = !phone
            findPreference<PreferenceCategory>("cat_phone")!!.isVisible = phone
        }

        private fun discover() {
            val ctx = requireContext()
            val waiting = AlertDialog.Builder(ctx).setMessage(R.string.setup_finding).show()
            thread(name = "homeapi-discovery") {
                val servers = LanDiscovery.find(ctx)
                activity?.runOnUiThread {
                    waiting.dismiss()
                    if (servers.isEmpty()) {
                        message(getString(R.string.setup_found_none))
                        return@runOnUiThread
                    }
                    AlertDialog.Builder(ctx)
                        .setTitle(R.string.pref_find)
                        .setItems(servers.map { "${it.name}\n${it.origin}" }.toTypedArray()) { _, i ->
                            findPreference<EditTextPreference>(ConfigStore.KEY_ADDRESS)!!.text = servers[i].origin.toString()
                        }
                        .show()
                }
            }
        }

        private fun showLog() {
            val ctx = requireContext()
            val text = LocalServer.logTail(ctx).ifBlank { "(empty)" }
            val view = TextView(ctx).apply {
                this.text = text
                setTextIsSelectable(true)
                typeface = android.graphics.Typeface.MONOSPACE
                textSize = 10f
                setPadding(32, 24, 32, 24)
            }
            AlertDialog.Builder(ctx)
                .setTitle(R.string.pref_log)
                .setView(android.widget.ScrollView(ctx).apply { addView(view) })
                .setPositiveButton(android.R.string.ok, null)
                .setNeutralButton(R.string.share_with) { _, _ ->
                    startActivity(Intent.createChooser(Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, text), null))
                }
                .show()
        }

        private fun message(text: String) {
            AlertDialog.Builder(requireContext()).setMessage(text).setPositiveButton(android.R.string.ok, null).show()
        }

        private companion object {
            const val KEY_RUN_ON_PHONE = "run_on_phone"
        }
    }
}
