package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Test

class NavigationPolicyTest {
    private val app = Origin.loopback(9999)

    private fun check(url: String, want: Navigation) = assertEquals(url, want, NavigationPolicy.decide(app, url))

    @Test
    fun theServersOwnPagesStayInTheApp() {
        check("http://127.0.0.1:9999/", Navigation.IN_APP)
        check("http://127.0.0.1:9999/api/entries/city", Navigation.IN_APP)
        check("blob:http://127.0.0.1:9999/0b1c", Navigation.IN_APP)
        check("data:image/png;base64,AAAA", Navigation.IN_APP)
        check("about:blank", Navigation.IN_APP)
    }

    @Test
    fun otherSitesGoToThePhone() {
        check("https://accounts.google.com/o/oauth2/auth?x=1", Navigation.EXTERNAL)
        check("http://127.0.0.1:9998/", Navigation.EXTERNAL)
        check("https://127.0.0.1:9999/", Navigation.EXTERNAL)
        check("mailto:someone@example.com", Navigation.EXTERNAL)
        check("tel:+15555550100", Navigation.EXTERNAL)
        check("vlc://127.0.0.1:9999/x", Navigation.EXTERNAL)
    }

    @Test
    fun intentUrlsAreTheirOwnCase() {
        check("intent://127.0.0.1:9999/x#Intent;scheme=http;type=video/*;package=org.videolan.vlc;end", Navigation.INTENT)
    }

    @Test
    fun dangerousSchemesAreBlocked() {
        check("javascript:alert(1)", Navigation.BLOCK)
        check("file:///data/data/io.github.chinmay28.homeapi/files/homeapi/homeapi.db", Navigation.BLOCK)
        check("content://com.android.contacts/contacts", Navigation.BLOCK)
        check("about:config", Navigation.BLOCK)
        check("weird", Navigation.BLOCK)
        check("", Navigation.BLOCK)
    }
}
