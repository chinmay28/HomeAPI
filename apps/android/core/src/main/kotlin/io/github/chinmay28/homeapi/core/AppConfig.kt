package io.github.chinmay28.homeapi.core

/** How the app reaches HomeAPI. */
enum class AppMode {
    /** A thin client: HomeAPI runs on another machine, and the app shows it. */
    REMOTE,

    /** Full stack: HomeAPI runs on this phone, and the app shows that. */
    ON_PHONE,
}

/**
 * Everything the person chose in Settings.
 *
 * Both modes' settings are kept side by side, so switching back and forth
 * keeps the other mode's address and port rather than starting over.
 */
data class AppConfig(
    /** Null until the first-run screen has been answered. */
    val mode: AppMode? = null,

    /** REMOTE: the server, as typed; [Origin.parse] reads it. */
    val serverAddress: String = "",

    /** ON_PHONE: the port the server on the phone listens on. */
    val localPort: Int = Origin.DEFAULT_PORT,

    /**
     * ON_PHONE: also serve other devices on this network — scripts, a
     * laptop's browser, a Minion display — rather than this phone alone.
     */
    val shareOnNetwork: Boolean = false,

    /**
     * ON_PHONE: keep the server running when the app is closed, so other
     * devices and the home-screen widget can still reach it.
     */
    val keepRunning: Boolean = false,

    /**
     * ON_PHONE: the name this phone answers network discovery with, which
     * other HomeAPI apps list it as. Blank means the phone's own name.
     */
    val deviceName: String = "",
) {
    val configured: Boolean get() = mode != null

    /** The origin the web view opens, or a reason it has none. */
    fun origin(): Result<Origin> = when (mode) {
        null -> Result.failure(IllegalStateException("Choose how to reach HomeAPI first."))
        AppMode.REMOTE -> Origin.parse(serverAddress)
        AppMode.ON_PHONE -> if (localPort in 1024..65535) {
            Result.success(Origin.loopback(localPort))
        } else {
            Result.failure(IllegalArgumentException("The port must be between 1024 and 65535."))
        }
    }

    /**
     * Whether going from this configuration to [next] needs the server on
     * the phone restarted: anything it was started with has changed.
     */
    fun serverChanged(next: AppConfig): Boolean =
        mode != next.mode ||
            (next.mode == AppMode.ON_PHONE &&
                (localPort != next.localPort || shareOnNetwork != next.shareOnNetwork || deviceName != next.deviceName))
}
