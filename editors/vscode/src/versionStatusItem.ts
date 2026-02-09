import * as vscode from "vscode";
import { Client } from "./client";
import { jsTsLanguageModes } from "./util";

export function setupVersionStatusItem(
    client: Client,
): vscode.Disposable[] {
    const statusItem = vscode.languages.createLanguageStatusItem("golar.version", jsTsLanguageModes);
    statusItem.name = "Golar version";
    statusItem.detail = "Golar version";
    return [
        statusItem,
        client.onStarted(() => {
            statusItem.text = client.getCurrentExe()!.version;
        }),
    ];
}
