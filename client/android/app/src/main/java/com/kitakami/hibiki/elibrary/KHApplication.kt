package com.kitakami.hibiki.elibrary

import android.app.Application

class KHApplication : Application() {

    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }
}
