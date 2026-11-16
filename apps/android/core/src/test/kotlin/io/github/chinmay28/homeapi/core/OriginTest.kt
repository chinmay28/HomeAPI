package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class OriginTest {

    private fun parsed(input: String) = Origin.parse(input).getOrThrow().toString()

    @Test
    fun aBareHostIsHomeApisDefaultPortOverHttp() {
        assertEquals("http://pi.local:9999", parsed("pi.local"))
        assertEquals("http://192.168.1.20:9999", parsed(" 192.168.1.20 "))
        assertEquals("http://192.168.1.20:9000", parsed("192.168.1.20:9000"))
    }

    @Test
    fun aSchemeSpelledOutKeepsItsOwnDefaultPort() {
        assertEquals("http://pi.local", parsed("http://pi.local"))
        assertEquals("https://home.example.com", parsed("https://home.example.com"))
        assertEquals("https://home.example.com:8443", parsed("HTTPS://Home.Example.com:8443/"))
    }

    @Test
    fun pathQueryAndFragmentAreDropped() {
        assertEquals("https://home.example.com", parsed("https://home.example.com/some/folder?x=1#y"))
    }

    @Test
    fun ipv6IsBracketed() {
        val o = Origin.parse("http://[fd00::20]:9999").getOrThrow()
        assertEquals("fd00::20", o.host)
        assertEquals("http://[fd00::20]:9999", o.toString())
        assertEquals("http://[fd00::20]:9999/api/health", o.url("api/health"))
    }

    @Test
    fun badAddressesAreRefusedWithAReason() {
        for (bad in listOf("", "   ", "ftp://pi.local", "https://user:pw@home.example.com", "http://", "http://pi.local:99999")) {
            assertTrue("accepted $bad", Origin.parse(bad).isFailure)
        }
        assertTrue(Origin.parse("").exceptionOrNull()!!.message!!.isNotBlank())
    }

    @Test
    fun containsIsSameOrigin() {
        val o = Origin.parse("http://pi.local:9999").getOrThrow()
        assertTrue(o.contains("http://pi.local:9999/api/entries?category=x"))
        assertTrue(o.contains("http://PI.local:9999"))
        assertFalse(o.contains("http://pi.local:9998/"))
        assertFalse(o.contains("https://pi.local:9999/"))
        assertFalse(o.contains("http://pi.local.evil.test:9999/"))
        assertFalse(o.contains("http://evil.test/?http://pi.local:9999"))
        assertFalse(o.contains("not a url"))
    }

    @Test
    fun ofReadsDefaultPorts() {
        assertEquals(Origin("https", "a.test", 443), Origin.of("https://a.test/x"))
        assertEquals(Origin("http", "a.test", 80), Origin.of("http://a.test"))
        assertNull(Origin.of("blob:http://a.test/123"))
        assertNull(Origin.of("mailto:me@a.test"))
    }

    @Test
    fun loopbackIsTheFullStackServer() {
        assertEquals("http://127.0.0.1:9999", Origin.loopback(9999).toString())
    }
}
