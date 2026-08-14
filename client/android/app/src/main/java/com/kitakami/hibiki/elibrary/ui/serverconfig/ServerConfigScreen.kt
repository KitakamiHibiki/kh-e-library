package com.kitakami.hibiki.elibrary.ui.serverconfig

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R

/**
 * Server root URL configuration. Rendered as the whole app until a valid root
 * URL is saved (the "gate"); also reachable from the bookshelf settings to
 * change the server later.
 */
@Composable
fun ServerConfigScreen(
    container: AppContainer,
    onDone: () -> Unit,
) {
    val viewModel: ServerConfigViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        ServerConfigViewModel(app.container)
    }

    val urlInvalid = viewModel.url.isNotBlank() && !viewModel.isUrlValid()

    Scaffold { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 28.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = stringResource(R.string.server_config_title),
                style = MaterialTheme.typography.headlineMedium,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(R.string.server_config_required),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(24.dp))
            OutlinedTextField(
                value = viewModel.url,
                onValueChange = viewModel::onUrlChange,
                label = { Text(stringResource(R.string.server_config_hint)) },
                placeholder = { Text(stringResource(R.string.server_config_example)) },
                singleLine = true,
                isError = urlInvalid || viewModel.testOutcome is TestOutcome.Failed,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))
            when (val outcome = viewModel.testOutcome) {
                TestOutcome.Idle -> {
                    if (urlInvalid) {
                        Text(
                            text = stringResource(R.string.server_config_invalid),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.error,
                        )
                    }
                }
                TestOutcome.Ok -> {
                    Text(
                        text = stringResource(R.string.server_config_ok),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
                is TestOutcome.Failed -> {
                    Text(
                        text = outcome.reason,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
            Spacer(Modifier.height(24.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                OutlinedButton(
                    onClick = viewModel::test,
                    enabled = !viewModel.testing && !viewModel.saving && viewModel.isUrlValid(),
                ) {
                    if (viewModel.testing) {
                        CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text(stringResource(R.string.server_config_test))
                    }
                }
                Spacer(Modifier.width(12.dp))
                Button(
                    onClick = { viewModel.save(onDone) },
                    enabled = !viewModel.saving && viewModel.isUrlValid(),
                ) {
                    if (viewModel.saving) {
                        CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text(stringResource(R.string.server_config_save))
                    }
                }
            }
        }
    }
}
