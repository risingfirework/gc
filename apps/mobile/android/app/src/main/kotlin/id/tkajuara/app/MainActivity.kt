package id.tkajuara.app

import android.view.WindowManager
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val channelName = "id.tkajuara.app/security"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName).setMethodCallHandler { call, result ->
            runOnUiThread {
                when (call.method) {
                    "enableSecureScreen" -> {
                        window.addFlags(WindowManager.LayoutParams.FLAG_SECURE)
                        result.success(null)
                    }
                    "disableSecureScreen" -> {
                        window.clearFlags(WindowManager.LayoutParams.FLAG_SECURE)
                        result.success(null)
                    }
                    else -> result.notImplemented()
                }
            }
        }
    }
}
