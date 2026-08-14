package com.kitakami.hibiki.elibrary.ui.tags

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Tag
import com.kitakami.hibiki.elibrary.ui.theme.BrandRed

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TagsScreen(
    container: AppContainer,
    onBack: () -> Unit,
) {
    val viewModel: TagsViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        TagsViewModel(app.container)
    }
    val context = LocalContext.current
    val snackbarHostState = remember { SnackbarHostState() }

    var dialogTag by remember { mutableStateOf<Tag?>(null) }   // non-null → edit dialog
    var createDialog by remember { mutableStateOf(false) }
    var deleteTag by remember { mutableStateOf<Tag?>(null) }

    LaunchedEffect(Unit) { viewModel.load() }

    LaunchedEffect(viewModel.message) {
        val msg = viewModel.message ?: return@LaunchedEffect
        snackbarHostState.showSnackbar(context.getString(msg.resId, *msg.args.toTypedArray()))
        viewModel.consumeMessage()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.tags_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.reader_back))
                    }
                },
                actions = {
                    IconButton(onClick = { createDialog = true }) {
                        Icon(Icons.Default.Add, contentDescription = stringResource(R.string.tags_new))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        if (viewModel.loading && viewModel.tags.isEmpty()) {
            Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        } else if (viewModel.tags.isEmpty()) {
            Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text(stringResource(R.string.common_empty), color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        } else {
            LazyColumn(Modifier.fillMaxSize().padding(innerPadding)) {
                items(viewModel.tags, key = { it.id }) { tag ->
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp, vertical = 6.dp),
                    ) {
                        Text(
                            text = "${tag.name} (${tag.count})",
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.weight(1f),
                        )
                        IconButton(onClick = { dialogTag = tag }) {
                            Icon(Icons.Default.Edit, contentDescription = stringResource(R.string.tags_rename))
                        }
                        IconButton(onClick = { deleteTag = tag }) {
                            Icon(
                                Icons.Default.Delete,
                                contentDescription = stringResource(R.string.common_delete),
                                tint = BrandRed,
                            )
                        }
                    }
                }
            }
        }
    }

    // Dialogs.
    if (createDialog) {
        TagNameDialog(
            title = stringResource(R.string.tags_new),
            onDismiss = { createDialog = false },
            onConfirm = { name ->
                viewModel.create(name)
                createDialog = false
            },
        )
    }
    dialogTag?.let { tag ->
        TagNameDialog(
            title = stringResource(R.string.tags_rename),
            initialName = tag.name,
            onDismiss = { dialogTag = null },
            onConfirm = { name ->
                viewModel.rename(tag.id, name)
                dialogTag = null
            },
        )
    }
    deleteTag?.let { tag ->
        AlertDialog(
            onDismissRequest = { deleteTag = null },
            title = { Text(stringResource(R.string.common_delete)) },
            text = { Text(stringResource(R.string.tags_delete_confirm, tag.name)) },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.delete(tag.id)
                    deleteTag = null
                }) {
                    Text(stringResource(R.string.common_delete), color = BrandRed)
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteTag = null }) {
                    Text(stringResource(R.string.common_cancel))
                }
            },
        )
    }
}

@Composable
private fun TagNameDialog(
    title: String,
    onDismiss: () -> Unit,
    onConfirm: (String) -> Unit,
    initialName: String = "",
) {
    var name by remember { mutableStateOf(initialName) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            OutlinedTextField(
                value = name,
                onValueChange = { name = it },
                placeholder = { Text(stringResource(R.string.tags_name)) },
                singleLine = true,
            )
        },
        confirmButton = {
            TextButton(onClick = { if (name.isNotBlank()) onConfirm(name.trim()) }, enabled = name.isNotBlank()) {
                Text(stringResource(R.string.common_confirm))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.common_cancel))
            }
        },
    )
}
