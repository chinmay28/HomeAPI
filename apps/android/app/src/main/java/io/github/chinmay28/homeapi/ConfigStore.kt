package io.github.chinmay28.homeapi

import android.content.Context
import android.content.SharedPreferences
import android.os.Build
import android.provider.Settings
import io.github.chinmay28.homeapi.core.AppConfig
import io.github.chinmay28.homeapi.core.AppMode
import io.github.chinmay28.homeapi.core.MachineName
import io.github.chinmay28.homeapi.core.Origin

/**
 * The settings, kept in the shared preferences the settings screen edits
 * directly — one store, so the screen and the rest of the app never disagree.
 */
class ConfigStore(context: Context) {
    private val app = context.applicationContext
    val prefs: SharedPreferences = app.getSharedPreferences(FILE, Context.MODE_PRIVATE)

    fun load(): AppConfig = AppConfig(
        mode = when (prefs.getString(KEY_MODE, null)) {
            MODE_REMOTE -> AppMode.REMOTE
            MODE_ON_PHONE -> AppMode.ON_PHONE
            else -> null
        },
        serverAddress = prefs.getString(KEY_ADDRESS, "") ?: "",
        localPort = prefs.getString(KEY_PORT, null)?.toIntOrNull() ?: Origin.DEFAULT_PORT,
        shareOnNetwork = prefs.getBoolean(KEY_SHARE, false),
        keepRunning = prefs.getBoolean(KEY_KEEP, false),
        deviceName = prefs.getString(KEY_NAME, "") ?: "",
    )

    fun save(c: AppConfig) {
        prefs.edit()
            .putString(KEY_MODE, c.mode?.let(::modeValue))
            .putString(KEY_ADDRESS, c.serverAddress)
            .putString(KEY_PORT, c.localPort.toString())
            .putBoolean(KEY_SHARE, c.shareOnNetwork)
            .putBoolean(KEY_KEEP, c.keepRunning)
            .putString(KEY_NAME, c.deviceName)
            .apply()
    }

    /** The name the server announces: the person's, else the phone's own. */
    fun deviceName(c: AppConfig = load()): String =
        c.deviceName.trim().ifEmpty { defaultDeviceName() }

    fun defaultDeviceName(): String = MachineName.of(
        runCatching { Settings.Global.getString(app.contentResolver, Settings.Global.DEVICE_NAME) }.getOrNull(),
        Build.MANUFACTURER,
        Build.MODEL,
    )

    companion object {
        const val FILE = "homeapi"
        const val KEY_MODE = "mode"
        const val KEY_ADDRESS = "server_address"
        const val KEY_PORT = "local_port"
        const val KEY_SHARE = "share_on_network"
        const val KEY_KEEP = "keep_running"
        const val KEY_NAME = "device_name"
        const val MODE_REMOTE = "remote"
        const val MODE_ON_PHONE = "on_phone"

        fun modeValue(m: AppMode) = if (m == AppMode.REMOTE) MODE_REMOTE else MODE_ON_PHONE
    }
}
