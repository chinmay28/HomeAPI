package io.github.chinmay28.homeapi.core

import org.json.JSONObject

/** Reading `GET /api/health`: `{"status":"ok","version":"v2026.11.30"}`. */
object Health {
    /** The server's version when [body] is a healthy answer, else null. */
    fun version(body: String): String? = try {
        val json = JSONObject(body)
        if (json.optString("status") == "ok") json.optString("version").ifBlank { "unknown" } else null
    } catch (_: Exception) {
        null
    }

    /**
     * Polls [probe] until it reports a version or [timeoutMs] passes, sleeping
     * [intervalMs] between tries. [probe] returns null while the server is not
     * answering yet and should not throw. [stillWanted] lets a caller give up
     * early — the server died, or the screen was closed.
     */
    fun await(
        timeoutMs: Long,
        intervalMs: Long = 250,
        clock: () -> Long = System::currentTimeMillis,
        sleep: (Long) -> Unit = Thread::sleep,
        stillWanted: () -> Boolean = { true },
        probe: () -> String?,
    ): String? {
        val deadline = clock() + timeoutMs
        while (stillWanted()) {
            probe()?.let { return it }
            if (clock() >= deadline) return null
            sleep(intervalMs)
        }
        return null
    }
}
