#
# from ultralytics import YOLO
#
# # 加载模型
# model = YOLO(r"E:\FyProject\Py\porject\pdsnvrAnalysis\models\best.pt")
#
# model.export(
#     format="engine",
#     imgsz=640,
#     batch=1,
#     end2end=False,
#     simplify=True,
#     device=0
# )
#
import torch
import tensorrt as trt
from ultralytics import YOLO

# 导出 ONNX
model = YOLO("best.pt")
model.export(
    format="onnx",
    imgsz=640,
    batch=1,
    end2end=False,
    simplify=True,
    device=0
)
onnx_path = "best.onnx"
engine_path = "best.engine"

#  ONNX → engine
TRT_LOGGER = trt.Logger(trt.Logger.INFO)
builder = trt.Builder(TRT_LOGGER)
network = builder.create_network(1 << int(trt.NetworkDefinitionCreationFlag.EXPLICIT_BATCH))
parser = trt.OnnxParser(network, TRT_LOGGER)

with open(onnx_path, "rb") as f:
    parser.parse(f.read())

config = builder.create_builder_config()
# config.set_flag(trt.BuilderFlag.FP16)  # GPU 加速
config.set_memory_pool_limit(trt.MemoryPoolType.WORKSPACE, 8 * 1024 * 1024 * 1024)

# 构建并保存 engine
serialized_engine = builder.build_serialized_network(network, config)
with open(engine_path, "wb") as f:
    f.write(serialized_engine)

print("engine导出完成！")
