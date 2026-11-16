package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AppConfigTest {
    @Test
    fun unconfiguredHasNoOrigin() {
        assertFalse(AppConfig().configured)
        assertTrue(AppConfig().origin().isFailure)
    }

    @Test
    fun remoteOpensTheTypedServer() {
        val c = AppConfig(mode = AppMode.REMOTE, serverAddress = "pi.local")
        assertEquals("http://pi.local:9999", c.origin().getOrThrow().toString())
        assertTrue(AppConfig(mode = AppMode.REMOTE, serverAddress = "").origin().isFailure)
    }

    @Test
    fun onPhoneOpensLoopback() {
        assertEquals("http://127.0.0.1:9999", AppConfig(mode = AppMode.ON_PHONE).origin().getOrThrow().toString())
        val c = AppConfig(mode = AppMode.ON_PHONE, localPort = 8200, serverAddress = "pi.local")
        assertEquals("http://127.0.0.1:8200", c.origin().getOrThrow().toString())
        assertTrue(c.copy(localPort = 80).origin().isFailure)
    }

    @Test
    fun serverChangedOnlyForWhatTheServerIsStartedWith() {
        val phone = AppConfig(mode = AppMode.ON_PHONE)
        assertFalse(phone.serverChanged(phone.copy(keepRunning = true)))
        assertFalse(phone.serverChanged(phone.copy(serverAddress = "x")))
        assertTrue(phone.serverChanged(phone.copy(localPort = 9000)))
        assertTrue(phone.serverChanged(phone.copy(shareOnNetwork = true)))
        assertTrue(phone.serverChanged(phone.copy(deviceName = "pocket")))
        assertTrue(phone.serverChanged(phone.copy(mode = AppMode.REMOTE)))
        val remote = AppConfig(mode = AppMode.REMOTE)
        assertFalse(remote.serverChanged(remote.copy(localPort = 9000)))
    }
}
