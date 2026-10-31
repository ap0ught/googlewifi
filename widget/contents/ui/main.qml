import QtQuick 2.15
import QtQuick.Controls 2.15 as Controls
import QtQuick.Layouts 1.15
import org.kde.plasma.plasmoid 2.0
import org.kde.plasma.plasma5support 2.0 as Plasma5Support
import org.kde.plasma.components 3.0 as PlasmaComponents

PlasmoidItem {
    id: root
    preferredRepresentation: compactRepresentation
    property var report: ({})
    property string selectedHost: ""
    property string command: "googlewifimon --json --ping-count 3"
    property string errorText: ""

    Plasma5Support.DataSource {
        id: commandSource
        engine: "executable"
        connectedSources: []

        function refresh() {
            connectSource(root.command)
        }

        onNewData: function(sourceName, data) {
            if (sourceName !== root.command) return
            var raw = data["stdout"] || ""
            try {
                var text = String(raw).trim()
                var parsed = JSON.parse(text)
                root.report = parsed
                root.errorText = ""
                if (root.selectedHost === "" && root.report.hosts)
                    root.selectedHost = Object.keys(root.report.hosts)[0]
                graph.requestPaint()
            } catch (e) {
                if (!root.report.hosts)
                    root.errorText = "Monitor output could not be parsed"
            }
            disconnectSource(sourceName)
        }
    }

    Timer {
        interval: 60000
        running: true
        repeat: true
        onTriggered: commandSource.refresh()
    }

    Component.onCompleted: commandSource.refresh()

    compactRepresentation: Item {
        implicitWidth: 110
        implicitHeight: 28

        PlasmaComponents.Label {
            anchors.fill: parent
            text: root.selectedStatus()
            color: root.selectedColor()
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
        }

        MouseArea {
            anchors.fill: parent
            onClicked: Plasmoid.expanded = true
        }
    }

    fullRepresentation: Rectangle {
        color: "#202124"
        implicitWidth: 540
        implicitHeight: 360
        radius: 8

        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 14
            spacing: 8

            RowLayout {
                Layout.fillWidth: true
                Controls.Label {
                    text: "Google Wifi"
                    font.bold: true
                    font.pointSize: 14
                    color: "#f8f9fa"
                }
                Item { Layout.fillWidth: true }
                PlasmaComponents.Label {
                    text: root.selectedStatus()
                    color: root.selectedColor()
                }
                PlasmaComponents.Label { text: root.lastUpdated() }
            }

            Controls.ComboBox {
                id: hostSelector
                Layout.fillWidth: true
                model: root.report.hosts ? Object.keys(root.report.hosts) : []
                currentIndex: Math.max(0, model.indexOf(root.selectedHost))
                onActivated: root.selectedHost = currentText
            }

            PlasmaComponents.Label {
                visible: root.errorText !== "" && !root.report.hosts
                text: root.errorText
                color: "#f28b82"
            }

            Canvas {
                id: graph
                Layout.fillWidth: true
                Layout.fillHeight: true
                onPaint: {
                    var ctx = getContext("2d")
                    ctx.reset()
                    ctx.fillStyle = "#292a2d"
                    ctx.fillRect(0, 0, width, height)
                    var points = root.samples()
                    if (!points.length) return
                    var max = 10
                    for (var i = 0; i < points.length; i++) max = Math.max(max, points[i].ping_avg_ms || 0)
                    ctx.strokeStyle = "#8ab4f8"
                    ctx.lineWidth = 2
                    ctx.beginPath()
                    for (var j = 0; j < points.length; j++) {
                        var x = points.length === 1 ? 0 : j * width / (points.length - 1)
                        var y = height - ((points[j].ping_avg_ms || 0) / max) * (height - 12) - 6
                        if (j === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y)
                    }
                    ctx.stroke()
                    ctx.fillStyle = "#bdc1c6"
                    ctx.font = "12px sans-serif"
                    ctx.fillText("latency (ms), last " + points.length + " checks", 8, 16)
                    ctx.fillText(max.toFixed(0) + " ms", 8, height - 6)
                    ctx.fillText("0 ms", width - 40, height - 6)
                }
            }

            PlasmaComponents.Label {
                Layout.fillWidth: true
                text: root.selectedReason()
                color: "#bdc1c6"
                elide: Text.ElideRight
            }
        }
    }

    function selectedHostState() {
        return report.hosts && selectedHost ? report.hosts[selectedHost] : null
    }
    function selectedStatus() {
        var s = selectedHostState()
        return s ? s.state : "UNKNOWN"
    }
    function selectedColor() {
        var s = selectedStatus()
        if (s === "OK") return "#81c995"
        if (s === "WARNING") return "#fdd663"
        if (s === "CRITICAL") return "#f28b82"
        return "#bdc1c6"
    }
    function selectedReason() {
        var s = selectedHostState()
        return s ? s.reason + " · " + s.consecutive + " consecutive checks" : "Waiting for monitor"
    }
    function samples() {
        return report.samples && selectedHost ? (report.samples[selectedHost] || []) : []
    }
    function lastUpdated() {
        var s = selectedHostState()
        return s && s.last_check ? new Date(s.last_check).toLocaleTimeString() : ""
    }
}
