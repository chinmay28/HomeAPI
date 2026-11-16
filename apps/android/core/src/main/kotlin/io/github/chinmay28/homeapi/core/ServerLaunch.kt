package io.github.chinmay28.homeapi.core

import java.io.File

/**
 * How HomeAPI on the phone is started: the command line and environment for
 * the homeapi binary, worked out from the settings and the app's directories.
 *
 * Kept apart from starting it so that what the server is told can be read,
 * and tested, without a phone. Every flag and variable here is one the server
 * documents (README, "Configuration"); none of them exists only for Android.
 */
data class ServerLaunch(
    /** The homeapi binary, packaged as libhomeapi.so so Android installs it executable. */
    val binary: File,

    /** Private to the app: the database lives here. */
    val dataDir: File,

    /** Private scratch space, which Android may clear. */
    val cacheDir: File,

    val port: Int,
    val shareOnNetwork: Boolean,

    /** What other HomeAPI apps list this phone as when they look for servers. */
    val machineName: String,

    /** An IANA zone name, e.g. "Europe/Berlin". */
    val timeZone: String,
) {
    val database: File get() = File(dataDir, "homeapi.db")

    fun command(): List<String> = listOf(
        binary.absolutePath,
        "--port", port.toString(),
        // Loopback unless asked: then only this phone's apps can reach it.
        "--host", if (shareOnNetwork) "0.0.0.0" else "127.0.0.1",
        // Answering discovery only makes sense on a server others can reach
        // (and the server never answers on a loopback bind anyway).
        "--discovery-port", if (shareOnNetwork) DiscoveryProtocol.PORT.toString() else "0",
    )

    fun environment(): Map<String, String> = buildMap {
        // Go reads these for the home directory and temporary files; on
        // Android it would otherwise pick /sdcard and /data/local/tmp, which
        // the app cannot write.
        put("HOME", dataDir.absolutePath)
        put("TMPDIR", cacheDir.absolutePath)
        put("HOMEAPI_DB_PATH", database.absolutePath)
        // Go on Android starts in UTC; the Minion display draws local time.
        if (timeZone.isNotBlank()) put("HOMEAPI_TZ", timeZone)
        // Every Android app sees the hostname "localhost".
        if (machineName.isNotBlank()) put("HOMEAPI_NAME", machineName.trim())
    }
}

/** What a phone gets called when the person has not named it. */
object MachineName {
    /**
     * A name from the phone's own: the name the person gave the device, else
     * its maker and model ("Pixel 9"), else "phone". Reduced to what reads
     * well in a list and travels in an environment variable.
     */
    fun of(deviceName: String?, manufacturer: String?, model: String?): String {
        val chosen = deviceName?.takeIf { it.isNotBlank() }
            ?: listOfNotNull(manufacturer?.takeIf { it.isNotBlank() }, model?.takeIf { it.isNotBlank() })
                .let { parts ->
                    // "Google Pixel 9" reads oddly when the model already says it.
                    if (parts.size == 2 && parts[1].startsWith(parts[0], ignoreCase = true)) parts[1] else parts.joinToString(" ")
                }
                .takeIf { it.isNotBlank() }
            ?: "phone"
        val cleaned = chosen.filter { !it.isISOControl() }.trim().replace(Regex("\\s+"), " ")
        return cleaned.take(48).ifBlank { "phone" }
    }
}
