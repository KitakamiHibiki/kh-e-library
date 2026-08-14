package com.kitakami.hibiki.elibrary

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import com.kitakami.hibiki.elibrary.ui.AppRoot
import com.kitakami.hibiki.elibrary.ui.theme.KHELibraryTheme

class MainActivity : ComponentActivity() {

    private val container: AppContainer
        get() = (application as KHApplication).container

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            KHELibraryTheme {
                AppRoot(container)
            }
        }
    }
}
