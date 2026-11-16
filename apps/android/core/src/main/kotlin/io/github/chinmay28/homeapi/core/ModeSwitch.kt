package io.github.chinmay28.homeapi.core

/** What happens when the person flips the app between its two modes. */
sealed interface ModeChange {
    /** Nothing stands in the way: save the new mode. */
    data object Apply : ModeChange

    /** HomeAPI cannot run here: this build or this processor has no server. */
    data object Unavailable : ModeChange

    /**
     * Thin client with no usable server address yet. The mode is still
     * saved — the other mode's settings are kept either way — but the
     * address has to be asked for before the app can open anything.
     */
    data object NeedsAddress : ModeChange
}

/**
 * The one decision behind the mode switch in Settings, kept here so it is
 * the same wherever the switch is offered and can be tested without Android.
 */
object ModeSwitch {
    fun check(target: AppMode, config: AppConfig, canRunHere: Boolean): ModeChange = when (target) {
        AppMode.ON_PHONE -> if (canRunHere) ModeChange.Apply else ModeChange.Unavailable
        AppMode.REMOTE -> if (Origin.parse(config.serverAddress).isSuccess) ModeChange.Apply else ModeChange.NeedsAddress
    }
}
