package com.kitakami.hibiki.elibrary.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.ui.bookdetail.BookDetailScreen
import com.kitakami.hibiki.elibrary.ui.bookshelf.BookShelfScreen
import com.kitakami.hibiki.elibrary.ui.reader.ReaderHostScreen
import com.kitakami.hibiki.elibrary.ui.serverconfig.ServerConfigScreen
import com.kitakami.hibiki.elibrary.ui.settings.SettingsScreen
import com.kitakami.hibiki.elibrary.ui.tags.TagsScreen

/**
 * App root.
 *
 * Enforces the "server root URL" gate: until a root URL is configured, the whole
 * app is the configuration screen and nothing else is reachable.
 */
@Composable
fun AppRoot(container: AppContainer) {
    val baseUrl by container.baseUrl.collectAsStateWithLifecycle()

    if (baseUrl.isBlank()) {
        // Gate — no server configured yet.
        ServerConfigScreen(container = container, onDone = {})
    } else {
        MainNavHost(container)
    }
}

private const val SHELF = "shelf"
private const val CONFIG = "config"
private const val BOOK = "book/{bookId}"
private const val READER = "reader/{bookId}"
private const val SETTINGS = "settings"
private const val TAGS = "tags"
private const val ARG_BOOK_ID = "bookId"

@Composable
private fun MainNavHost(container: AppContainer) {
    val navController = rememberNavController()

    NavHost(navController = navController, startDestination = SHELF) {
        composable(SHELF) {
            BookShelfScreen(
                container = container,
                onOpenBook = { id -> navController.navigate("book/$id") },
                onOpenSettings = { navController.navigate(SETTINGS) },
                onOpenTags = { navController.navigate(TAGS) },
            )
        }
        composable(CONFIG) {
            ServerConfigScreen(
                container = container,
                onDone = { navController.popBackStack() },
            )
        }
        composable(SETTINGS) {
            SettingsScreen(
                container = container,
                onBack = { navController.popBackStack() },
                onOpenServerConfig = { navController.navigate(CONFIG) },
                onOpenTags = { navController.navigate(TAGS) },
            )
        }
        composable(TAGS) {
            TagsScreen(
                container = container,
                onBack = { navController.popBackStack() },
            )
        }
        composable(
            route = BOOK,
            arguments = listOf(navArgument(ARG_BOOK_ID) { type = NavType.LongType }),
        ) { entry ->
            val bookId = entry.arguments?.getLong(ARG_BOOK_ID) ?: return@composable
            BookDetailScreen(
                container = container,
                bookId = bookId,
                onBack = { navController.popBackStack() },
                onRead = { id -> navController.navigate("reader/$id") },
            )
        }
        composable(
            route = READER,
            arguments = listOf(navArgument(ARG_BOOK_ID) { type = NavType.LongType }),
        ) { entry ->
            val bookId = entry.arguments?.getLong(ARG_BOOK_ID) ?: return@composable
            ReaderHostScreen(
                container = container,
                bookId = bookId,
                onBack = { navController.popBackStack() },
                onFinished = { navController.popBackStack() },
            )
        }
    }
}
