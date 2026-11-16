package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Test

class ModeSwitchTest {
    private val withServer = AppConfig(mode = AppMode.ON_PHONE, serverAddress = "pi.local")
    private val noServer = AppConfig(mode = AppMode.ON_PHONE, serverAddress = "")

    @Test
    fun onPhoneNeedsAServerInTheBuild() {
        assertEquals(ModeChange.Apply, ModeSwitch.check(AppMode.ON_PHONE, noServer, canRunHere = true))
        assertEquals(ModeChange.Unavailable, ModeSwitch.check(AppMode.ON_PHONE, noServer, canRunHere = false))
    }

    @Test
    fun remoteWithAKnownAddressApplies() {
        assertEquals(ModeChange.Apply, ModeSwitch.check(AppMode.REMOTE, withServer.copy(mode = AppMode.ON_PHONE), canRunHere = true))
        // Whether the phone could run a server has no bearing on leaving it.
        assertEquals(ModeChange.Apply, ModeSwitch.check(AppMode.REMOTE, withServer, canRunHere = false))
    }

    @Test
    fun remoteWithoutAnAddressAsksForOne() {
        assertEquals(ModeChange.NeedsAddress, ModeSwitch.check(AppMode.REMOTE, noServer, canRunHere = true))
        assertEquals(ModeChange.NeedsAddress, ModeSwitch.check(AppMode.REMOTE, noServer.copy(serverAddress = "   "), canRunHere = true))
    }
}
