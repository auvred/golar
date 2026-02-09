import * as vscode from "vscode";

export function setupStatusBar(): vscode.Disposable {
    const statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBarItem.text = "$(beaker) golar";
    statusBarItem.tooltip = "Golar Vue Language Server";
    statusBarItem.command = "golar.showMenu";
    statusBarItem.backgroundColor = new vscode.ThemeColor("statusBarItem.warningBackground");
    statusBarItem.show();
    return statusBarItem;
}
