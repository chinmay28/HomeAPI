package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.io.File

class DiscoveryProtocolTest {
    @Test
    fun readsAnAnnouncement() {
        // Exactly what internal/discovery sends.
        val f = DiscoveryProtocol.parse("""{"service":"homeapi","name":"raspberrypi","port":9999,"version":"v2026.11.30"}""", "/192.168.1.20")!!
        assertEquals("raspberrypi", f.name)
        assertEquals("http://192.168.1.20:9999", f.origin.toString())
        assertEquals("v2026.11.30", f.version)
    }

    @Test
    fun secureMeansHttpsAndAMissingNameIsTheHost() {
        val f = DiscoveryProtocol.parse("""{"service":"homeapi","port":443,"secure":true}""", "fd00::20%wlan0")!!
        assertEquals("https://[fd00::20]", f.origin.toString())
        assertEquals("fd00::20", f.name)
    }

    @Test
    fun ignoresAnythingElse() {
        assertNull(DiscoveryProtocol.parse("""{"service":"sand-vault","port":8123}""", "10.0.0.1"))
        assertNull(DiscoveryProtocol.parse("""{"service":"homeapi","port":0}""", "10.0.0.1"))
        assertNull(DiscoveryProtocol.parse("HOMEAPI-DISCOVER/1", "10.0.0.1"))
        assertNull(DiscoveryProtocol.parse("""{"service":"homeapi","port":9999}""", ""))
    }

    /** The constants are the server's; read them out of its Go source. */
    @Test
    fun matchesTheServer() {
        val go = generateSequence(File("").absoluteFile) { it.parentFile }
            .map { File(it, "internal/discovery/discovery.go") }
            .firstOrNull { it.isFile }
            ?: return // :core built outside the repository
        val src = go.readText()
        fun const(name: String) = Regex("""(?m)^const $name = (.+)$""").find(src)!!.groupValues[1].trim()
        assertEquals(DiscoveryProtocol.PORT.toString(), const("DefaultPort"))
        assertEquals("\"${DiscoveryProtocol.QUERY}\"", const("Query"))
        assertEquals("\"${DiscoveryProtocol.SERVICE}\"", const("Service"))
    }
}
