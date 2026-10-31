import QtQuick 2.15
import QtQuick.Controls 2.15 as Controls
import QtQuick.Layouts 1.15
import org.kde.plasma.plasmoid 2.0
import org.kde.plasma.plasma5support 2.0 as Plasma5Support
import org.kde.plasma.components 3.0 as PlasmaComponents

PlasmoidItem {
    id: root
    preferredRepresentation: fullRepresentation
    property var report: ({})
    property string selectedHost: ""
    property string command: "googlewifimon --json --ping-count 3"
    property string errorText: ""
    property int requestId: 0

    Plasma5Support.DataSource {
        id: commandSource
        engine: "executable"
        connectedSources: []

        function refresh() {
            root.requestId += 1
            connectedSources = [root.command + " # " + root.requestId]
        }

        onNewData: function(sourceName, data) {
            var raw = (data && data["stdout"] !== undefined) ? data["stdout"] : ""
            try {
                var text = String(raw).trim()
                var parsed = JSON.parse(text)
                root.report = parsed
                root.errorText = ""
                if (root.selectedHost === "" && root.report.hosts)
                    root.selectedHost = Object.keys(root.report.hosts)[0]
                graphRoot.requestPaint()
                graphMesh.requestPaint()
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

    compactRepresentation: PlasmaComponents.Label {
        text: root.selectedStatus()
        color: root.selectedColor()
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideRight
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

            PlasmaComponents.Label {
                visible: root.errorText !== "" && !root.report.hosts
                text: root.errorText
                color: "#f28b82"
            }

            ColumnLayout {
                Layout.fillWidth: true
                Layout.fillHeight: true
                spacing: 6

                PlasmaComponents.Label {
                    text: "192.168.86.1 · " + root.hostState("192.168.86.1")
                    color: root.hostColor("192.168.86.1")
                }
                Canvas {
                    id: graphRoot
                    property string host: "192.168.86.1"
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    onPaint: root.paintGraph(this, "192.168.86.1")
                }

                PlasmaComponents.Label {
                    text: "192.168.86.23 · " + root.hostState("192.168.86.23")
                    color: root.hostColor("192.168.86.23")
                }
                Canvas {
                    id: graphMesh
                    property string host: "192.168.86.23"
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    onPaint: root.paintGraph(this, "192.168.86.23")
                }
            }

            PlasmaComponents.Label {
                Layout.fillWidth: true
                text: root.overallReason()
                color: "#bdc1c6"
                elide: Text.ElideRight
            }
        }
    }

    function hostState(host) {
        var state = report.hosts && report.hosts[host]
        return state ? state.state : "WAITING"
    }
    function hostAt(index) {
        var hosts = report.hosts ? Object.keys(report.hosts) : []
        return hosts.length > index ? hosts[index] : ""
    }
    function hostTitle(host) {
        if (!host) return "Waiting for monitor data"
        var state = report.hosts[host]
        return host + " · " + (state ? state.state : "UNKNOWN")
    }
    function hostColor(host) {
        var state = report.hosts && host ? report.hosts[host] : null
        var value = state ? state.state : "UNKNOWN"
        if (value === "OK") return "#81c995"
        if (value === "WARNING") return "#fdd663"
        if (value === "CRITICAL") return "#f28b82"
        return "#bdc1c6"
    }
    function paintGraph(canvas, host) {
        var ctx = canvas.getContext("2d")
        ctx.reset()
        ctx.fillStyle = "#292a2d"
        ctx.fillRect(0, 0, canvas.width, canvas.height)
        var points = report.samples && host ? (report.samples[host] || []) : []
        if (!points.length) {
            ctx.fillStyle = "#bdc1c6"
            ctx.font = "12px sans-serif"
            ctx.fillText("waiting for samples", 8, 18)
            return
        }
        var max = 10
        for (var i = 0; i < points.length; i++) max = Math.max(max, points[i].ping_avg_ms || 0)
        ctx.strokeStyle = "#8ab4f8"
        ctx.lineWidth = 2
        ctx.beginPath()
        for (var j = 0; j < points.length; j++) {
            var x = points.length === 1 ? 0 : j * canvas.width / (points.length - 1)
            var y = canvas.height - ((points[j].ping_avg_ms || 0) / max) * (canvas.height - 12) - 6
            if (j === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y)
        }
        ctx.stroke()
        ctx.fillStyle = "#bdc1c6"
        ctx.font = "12px sans-serif"
        ctx.fillText("latency (ms), last " + points.length + " checks", 8, 16)
        ctx.fillText(max.toFixed(0) + " ms", 8, canvas.height - 6)
        ctx.fillText("0 ms", canvas.width - 40, canvas.height - 6)
    }
    function overallReason() {
        var hosts = report.hosts ? Object.keys(report.hosts) : []
        if (!hosts.length) return "Waiting for monitor"
        var parts = []
        for (var i = 0; i < hosts.length; i++) {
            var state = report.hosts[hosts[i]]
            parts.push(hosts[i] + ": " + state.reason)
        }
        return parts.join(" · ")
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
