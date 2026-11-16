package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class EntryWidgetTest {
    @Test
    fun keysAreTrimmedAndMustLookUp() {
        assertEquals("city", EntryWidget.key("  city ").getOrThrow())
        assertTrue(EntryWidget.key(" ").isFailure)
        assertTrue(EntryWidget.key("a/b").isFailure)
    }

    @Test
    fun pathEncodesTheKeyAsOneSegment() {
        assertEquals("/api/entries/city", EntryWidget.path("city"))
        assertEquals("/api/entries/my%20key%3F", EntryWidget.path("my key?"))
        assertEquals("/api/entries/caf%C3%A9", EntryWidget.path("café"))
    }

    @Test
    fun showsTheStoredTextByteForByte() {
        // What GET /api/entries/:key sends: value is parsed JSON, value_text the stored string.
        val body = """{"id":3,"category":"watchlist","key":"tickers","value":["AAPL","MSFT"],"value_text":"[\n  \"AAPL\",\n  \"MSFT\"\n]"}"""
        assertEquals(
            WidgetContent.Value("tickers", "watchlist", "[\n  \"AAPL\",\n  \"MSFT\"\n]"),
            EntryWidget.read("tickers", 200, body),
        )
    }

    @Test
    fun anOlderServerWithoutValueTextFallsBackToValue() {
        assertEquals(
            WidgetContent.Value("city", "default", "San Jose"),
            EntryWidget.read("city", 200, """{"key":"city","category":"default","value":{"data":"San Jose"}}"""),
        )
        val obj = EntryWidget.read("geo", 200, """{"key":"geo","value":{"lat":37.3}}""") as WidgetContent.Value
        assertEquals("""{"lat":37.3}""", obj.text)
    }

    @Test
    fun anIdLookupIsTitledWithTheRealKey() {
        val v = EntryWidget.read("12", 200, """{"key":"city","category":"default","value_text":"Sunnyvale"}""") as WidgetContent.Value
        assertEquals("city", v.key)
    }

    @Test
    fun missingAndBrokenAnswers() {
        assertEquals(WidgetContent.Missing("city"), EntryWidget.read("city", 404, """{"error":"entry not found"}"""))
        assertTrue(EntryWidget.read("city", 500, "") is WidgetContent.Problem)
        assertTrue(EntryWidget.read("city", 200, "<html>proxy</html>") is WidgetContent.Problem)
    }

    @Test
    fun longValuesAreClipped() {
        val long = "x".repeat(5_000)
        val v = EntryWidget.read("k", 200, """{"key":"k","value_text":"$long"}""") as WidgetContent.Value
        assertEquals(EntryWidget.MAX_CHARS, v.text.length)
        assertTrue(v.text.endsWith("…"))
    }
}
