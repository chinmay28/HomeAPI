package io.github.chinmay28.homeapi.core

import org.json.JSONArray
import org.json.JSONObject
import java.net.URLEncoder
import java.nio.charset.StandardCharsets

/** What the home-screen widget shows. */
sealed class WidgetContent {
    /** No key chosen yet. */
    data object NotSet : WidgetContent()

    /** The entry's value, as stored. [key] is the server's spelling of it. */
    data class Value(val key: String, val category: String, val text: String) : WidgetContent()

    /** The server answered, and has no such entry. */
    data class Missing(val key: String) : WidgetContent()

    /** The server could not be asked, or answered something unreadable. */
    data class Problem(val key: String, val message: String) : WidgetContent()
}

/**
 * The home-screen widget: one entry's value, pinned by key.
 *
 * It asks `GET /api/entries/<key>` — the same lookup a script does — and
 * shows `value_text`, the value byte-for-byte as stored, falling back to the
 * parsed `value` for a server too old to send it (the same rule as the web
 * GUI's valueText()). A key made only of digits is an entry id to the API,
 * which is why the widget titles itself with the key the server sends back.
 */
object EntryWidget {
    /** As much of a value as a widget could ever show. */
    const val MAX_CHARS = 1_000

    /** The key as stored in the widget's settings, or a reason it can't be one. */
    fun key(input: String): Result<String> {
        val k = input.trim()
        return when {
            k.isEmpty() -> Result.failure(IllegalArgumentException("Enter the key of an entry."))
            k.contains('/') -> Result.failure(IllegalArgumentException("A key with a / in it cannot be looked up by key."))
            else -> Result.success(k)
        }
    }

    /** Where the entry is, on the server's origin. */
    fun path(key: String): String =
        "/api/entries/" + URLEncoder.encode(key, StandardCharsets.UTF_8).replace("+", "%20")

    /** Reads the server's answer to [path]. */
    fun read(key: String, status: Int, body: String): WidgetContent = when (status) {
        200 -> try {
            val e = JSONObject(body)
            val text = if (e.opt("value_text") is String) e.getString("value_text") else display(e.opt("value"))
            WidgetContent.Value(e.optString("key").ifBlank { key }, e.optString("category"), clip(text))
        } catch (_: Exception) {
            WidgetContent.Problem(key, "The server's answer is not a HomeAPI entry.")
        }
        404 -> WidgetContent.Missing(key)
        else -> WidgetContent.Problem(key, "The server answered $status.")
    }

    /** The parsed `value` as text: a `{"data": …}` envelope is its string. */
    fun display(v: Any?): String = when {
        v == null || v == JSONObject.NULL -> ""
        v is JSONObject && v.length() == 1 && v.has("data") -> display(v.get("data"))
        v is JSONObject || v is JSONArray -> v.toString()
        else -> v.toString()
    }

    private fun clip(text: String): String {
        val t = text.trimEnd()
        return if (t.length <= MAX_CHARS) t else t.take(MAX_CHARS - 1).trimEnd() + "…"
    }
}
