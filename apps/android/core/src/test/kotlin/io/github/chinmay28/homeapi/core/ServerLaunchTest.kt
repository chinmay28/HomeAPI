package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test
import java.io.File

class ServerLaunchTest {
    private val launch = ServerLaunch(
        binary = File("/data/app/lib/arm64/libhomeapi.so"),
        dataDir = File("/data/user/0/io.github.chinmay28.homeapi/files/homeapi"),
        cacheDir = File("/data/user/0/io.github.chinmay28.homeapi/cache/homeapi"),
        port = 9999,
        shareOnNetwork = false,
        machineName = "Pixel 9",
        timeZone = "Europe/Berlin",
    )

    @Test
    fun loopbackByDefaultWithNoDiscovery() {
        assertEquals(
            listOf(
                "/data/app/lib/arm64/libhomeapi.so",
                "--port", "9999",
                "--host", "127.0.0.1",
                "--discovery-port", "0",
            ),
            launch.command(),
        )
    }

    @Test
    fun sharingBindsEveryInterfaceAndAnswersDiscovery() {
        val cmd = launch.copy(shareOnNetwork = true, port = 9000).command()
        assertEquals("0.0.0.0", cmd[cmd.indexOf("--host") + 1])
        assertEquals("9998", cmd[cmd.indexOf("--discovery-port") + 1])
        assertEquals("9000", cmd[cmd.indexOf("--port") + 1])
    }

    @Test
    fun environmentPointsTheServerAtTheAppsOwnDirectories() {
        val env = launch.environment()
        assertEquals("/data/user/0/io.github.chinmay28.homeapi/files/homeapi", env["HOME"])
        assertEquals("/data/user/0/io.github.chinmay28.homeapi/cache/homeapi", env["TMPDIR"])
        assertEquals("/data/user/0/io.github.chinmay28.homeapi/files/homeapi/homeapi.db", env["HOMEAPI_DB_PATH"])
        assertEquals("Europe/Berlin", env["HOMEAPI_TZ"])
        assertEquals("Pixel 9", env["HOMEAPI_NAME"])
    }

    @Test
    fun blankNameAndZoneAreLeftUnset() {
        val env = launch.copy(machineName = " ", timeZone = "").environment()
        assertFalse(env.containsKey("HOMEAPI_NAME"))
        assertFalse(env.containsKey("HOMEAPI_TZ"))
    }

    @Test
    fun machineNames() {
        assertEquals("Chinmay's phone", MachineName.of("Chinmay's phone", "Google", "Pixel 9"))
        assertEquals("Google Pixel 9", MachineName.of(null, "Google", "Pixel 9"))
        assertEquals("Samsung SM-S921B", MachineName.of("", "Samsung", "SM-S921B"))
        assertEquals("Google Pixel", MachineName.of(null, "Google", "Google Pixel"))
        assertEquals("phone", MachineName.of(null, null, null))
        assertEquals("a b", MachineName.of("a\n  b\u0007", null, null))
        assertEquals(48, MachineName.of("x".repeat(100), null, null).length)
    }
}
