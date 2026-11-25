// Real-Flutter twin of flutter-go's perf/perf_test.go grid benchmark.
//
// Builds the same 100×100 grid (plus a text label per row), calls setState
// every frame, and reports Flutter's own FrameTiming numbers. Run in profile
// mode:  flutter build macos --profile, then launch the binary directly.
// Pass --dart-define=LAYOUT=true to also change cell sizes every frame.
import 'dart:io';

import 'package:flutter/scheduler.dart';
import 'package:flutter/widgets.dart';

const bool layoutMode = bool.fromEnvironment('LAYOUT');
const int warmup = 60, measured = 300, rows = 100, cols = 100;

void main() => runApp(const Directionality(textDirection: TextDirection.ltr, child: Grid()));

class Grid extends StatefulWidget {
  const Grid({super.key});
  @override
  State<Grid> createState() => _GridState();
}

class _GridState extends State<Grid> {
  int tick = 0;
  final List<FrameTiming> timings = <FrameTiming>[];

  @override
  void initState() {
    super.initState();
    SchedulerBinding.instance.addTimingsCallback(_onTimings);
    SchedulerBinding.instance.addPersistentFrameCallback((_) {
      // Request the next frame with new data, like the Go benchmark.
      SchedulerBinding.instance.addPostFrameCallback((_) => setState(() => tick++));
    });
  }

  void _onTimings(List<FrameTiming> batch) {
    timings.addAll(batch);
    if (timings.length < warmup + measured) {
      return;
    }
    final List<FrameTiming> m = timings.sublist(warmup, warmup + measured);
    String stats(String name, Duration Function(FrameTiming) f) {
      final List<int> us = m.map((FrameTiming t) => f(t).inMicroseconds).toList()..sort();
      String p(double q) => '${(us[(q * (us.length - 1)).round()] / 1000).toStringAsFixed(2)}ms';
      return '$name p50 ${p(.5)}  p90 ${p(.9)}  p99 ${p(.99)}  max ${p(1)}';
    }

    int elements = 0;
    void count(Element e) {
      elements++;
      e.visitChildren(count);
    }

    WidgetsBinding.instance.rootElement!.visitChildren(count);
    stdout.writeln('flutter layout=$layoutMode: $elements elements');
    stdout.writeln('  ${stats('build+layout+paint (UI thread)', (FrameTiming t) => t.buildDuration)}');
    stdout.writeln('  ${stats('raster (GPU thread)', (FrameTiming t) => t.rasterDuration)}');
    exit(0);
  }

  @override
  Widget build(BuildContext context) {
    final double side = layoutMode && tick.isOdd ? 4 : 5;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        for (int i = 0; i < rows; i++)
          Row(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              SizedBox(width: 40, child: Text('$i', style: const TextStyle(fontSize: 4))),
              for (int j = 0; j < cols; j++)
                ColoredBox(
                  color: Color.from(alpha: 1, red: ((i + j + tick) % 256) / 255, green: 0.5, blue: 0.5),
                  child: SizedBox(width: side, height: side),
                ),
            ],
          ),
      ],
    );
  }
}
