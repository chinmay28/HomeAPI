package io.github.chinmay28.homeapi

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager

class HomeApiApp : Application() {
    override fun onCreate() {
        super.onCreate()
        val nm = getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_SERVER, getString(R.string.channel_server), NotificationManager.IMPORTANCE_LOW)
                .apply { setShowBadge(false) },
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_DOWNLOADS, getString(R.string.channel_downloads), NotificationManager.IMPORTANCE_LOW),
        )
    }

    companion object {
        const val CHANNEL_SERVER = "server"
        const val CHANNEL_DOWNLOADS = "downloads"
    }
}
