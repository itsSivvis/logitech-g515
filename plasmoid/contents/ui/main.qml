import QtQuick
import QtQuick.Layouts
import org.kde.plasma.plasmoid
import org.kde.plasma.core as PlasmaCore
import org.kde.plasma.plasma5support as Plasma5Support
import org.kde.kirigami as Kirigami

PlasmoidItem {
    id: root

    // g515-battery is looked up in PATH, ~/.local/bin and ~/bin
    readonly property string command: "g515-battery 2>/dev/null || \"$HOME/.local/bin/g515-battery\" 2>/dev/null || \"$HOME/bin/g515-battery\" 2>/dev/null || true"
    property int level: -1
    readonly property bool lowBattery: level >= 0 && level <= 20
    readonly property color textColor: lowBattery ? Kirigami.Theme.negativeTextColor : Kirigami.Theme.textColor

    preferredRepresentation: fullRepresentation
    toolTipMainText: "Logitech G515 battery"
    toolTipSubText: level >= 0 ? "Battery: " + level + " %" : "keyboard not reachable (install g515-battery)"

    Plasmoid.status: level >= 0 && lowBattery ? PlasmaCore.Types.NeedsAttentionStatus
                                               : PlasmaCore.Types.ActiveStatus

    function refresh() {
        if (command.length > 0) {
            exec.connectSource(command)
        }
    }

    Plasma5Support.DataSource {
        id: exec
        engine: "executable"
        connectedSources: []
        onNewData: function(sourceName, data) {
            disconnectSource(sourceName)
            let out = data["stdout"] !== undefined ? String(data["stdout"]) : ""
            if (out.length === 0) {
                for (const k in data) {
                    if (typeof data[k] === "string" && /\d/.test(data[k])) {
                        out = data[k]
                        break
                    }
                }
            }
            const m = out.match(/(\d{1,3})/)
            root.level = m ? Math.min(100, parseInt(m[1], 10)) : -1
        }
    }

    Timer {
        interval: 30000
        running: true
        repeat: true
        onTriggered: root.refresh()
    }

    Component.onCompleted: root.refresh()

    fullRepresentation: RowLayout {
        spacing: Kirigami.Units.smallSpacing

        Kirigami.Icon {
            source: "input-keyboard-symbolic"
            Layout.preferredWidth: Kirigami.Units.iconSizes.small
            Layout.preferredHeight: Kirigami.Units.iconSizes.small
            Layout.alignment: Qt.AlignVCenter
            color: root.textColor
        }

        Text {
            text: root.level >= 0 ? "G515 " + root.level + "%" : "G515 --"
            color: root.textColor
            font: Kirigami.Theme.defaultFont
            Layout.alignment: Qt.AlignVCenter
        }
    }

    compactRepresentation: Text {
        text: root.level >= 0 ? root.level + "%" : "--"
        color: root.textColor
        font: Kirigami.Theme.defaultFont
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
    }
}
