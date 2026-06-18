# #
# import time
# import cv2
# import grpc
# import numpy as np
# import torch
# import tensorrt as trt
# from concurrent import futures
# from threading import Lock
#
# import analysis_pb2
# import analysis_pb2_grpc
# # 全局配置
# ENGINE_PATH = "best.engine"
# INPUT_W = 640
# INPUT_H = 640
# CONF_THRESH = 0.7
# NMS_THRESH = 0.75
# # 上下文数量
# CONTEXT_NUM = 8
# TRT_LOGGER = trt.Logger(trt.Logger.ERROR)  # 关闭冗余日志降延迟
#
# #  TensorRT 多路优化池
# class TensorRTInfer:
#     def __init__(self, engine_path):
#         with open(engine_path, "rb") as f, trt.Runtime(TRT_LOGGER) as runtime:
#             self.engine = runtime.deserialize_cuda_engine(f.read())
#
#         self.contexts = []
#         self.streams = []
#         self.lock = Lock()
#
#         # 预创建独立上下文+独立CUDA流，互不抢占
#         for _ in range(CONTEXT_NUM):
#             ctx = self.engine.create_execution_context()
#             stream = torch.cuda.Stream()
#             self.contexts.append(ctx)
#             self.streams.append(stream)
#
#         self.input_name = self.engine.get_tensor_name(0)
#         self.output_name = self.engine.get_tensor_name(1)
#
#     def get_resource(self):
#         """取出一组空闲上下文+流"""
#         with self.lock:
#             ctx = self.contexts.pop()
#             stream = self.streams.pop()
#         return ctx, stream
#
#     def release_resource(self, ctx, stream):
#         """归还资源"""
#         with self.lock:
#             self.contexts.append(ctx)
#             self.streams.append(stream)
#
#     def infer(self, input_array):
#         ctx, stream = self.get_resource()
#         try:
#             with torch.cuda.stream(stream):
#                 # 非阻塞上传，降低延迟  # Orin NX 必须去掉 non_blocking，否则报错
#                 inp = torch.from_numpy(input_array).cuda(non_blocking=True).contiguous()
#                 ctx.set_tensor_address(self.input_name, inp.data_ptr())
#
#                 out_shape = tuple(ctx.get_tensor_shape(self.output_name))
#                 out = torch.empty(out_shape, dtype=torch.float32, device='cuda').contiguous()
#                 ctx.set_tensor_address(self.output_name, out.data_ptr())
#
#                 ctx.execute_async_v3(stream.cuda_stream)
#                 stream.synchronize()
#                 return out.cpu().numpy()
#         finally:
#             self.release_resource(ctx, stream)
#
# trt_model = TensorRTInfer(ENGINE_PATH)
#
# # 极速预处理
# def preprocess(image):
#     img = cv2.cvtColor(image, cv2.COLOR_RGBA2BGR)
#     img = cv2.resize(img, (INPUT_W, INPUT_H), interpolation=cv2.INTER_LINEAR)
#     img = img.astype(np.float32) / 255.0
#     img = np.transpose(img, (2, 0, 1))
#     return np.expand_dims(img, axis=0)
#
# # 后处理
# def postprocess(output, orig_w, orig_h):
#     output = output.squeeze(0).transpose(1, 0)
#     boxes = output[:, :4]
#     scores = output[:, 4:]
#     max_scores = np.max(scores, axis=1)
#     class_ids = np.argmax(scores, axis=1)
#
#     keep = max_scores >= CONF_THRESH
#     boxes = boxes[keep]
#     max_scores = max_scores[keep]
#     class_ids = class_ids[keep]
#
#     if len(boxes) == 0:
#         return []
#
#     x1 = boxes[:, 0] - boxes[:, 2] / 2
#     y1 = boxes[:, 1] - boxes[:, 3] / 2
#     x2 = boxes[:, 0] + boxes[:, 2] / 2
#     y2 = boxes[:, 1] + boxes[:, 3] / 2
#
#     x1 *= orig_w / INPUT_W
#     y1 *= orig_h / INPUT_H
#     x2 *= orig_w / INPUT_W
#     y2 *= orig_h / INPUT_H
#
#     nms_boxes = np.stack([x1, y1, x2-x1, y2-y1], axis=1)
#     indices = cv2.dnn.NMSBoxes(nms_boxes.tolist(), max_scores.tolist(), CONF_THRESH, NMS_THRESH)
#
#     res = []
#     for i in indices:
#         i = i[0] if isinstance(i, (list, np.ndarray)) else i
#         res.append(([x1[i], y1[i], x2[i], y2[i]], max_scores[i], class_ids[i]))
#     return res
#
# # # COCO 类别
# # YOLO_CLASSES = [
# #     "person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
# #     "traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat", "dog",
# #     "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella",
# #     "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball", "kite",
# #     "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket", "bottle",
# #     "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich",
# #     "orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch",
# #     "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse", "remote",
# #     "keyboard", "cell phone", "microwave", "oven", "toaster", "sink", "refrigerator", "book",
# #     "clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush"
# # ]
#
# YOLO_CLASSES = ["person","car"]
#
# # gRPC 服务
# class InferService(analysis_pb2_grpc.InferServiceServicer):
#     def Infer(self, request, context):
#         try:
#             data = np.frombuffer(request.image_data, np.uint8)
#             img = data.reshape((request.height, request.width, 4))
#             orig_w = request.width
#             orig_h = request.height
#
#             out = trt_model.infer(preprocess(img))
#             dets = postprocess(out, orig_w, orig_h)
#
#             response = analysis_pb2.InferResponse()
#             has_person = False
#
#             for box, score, cid in dets:
#                 label = YOLO_CLASSES[int(cid)]
#                 b = response.boxes.add()
#                 b.label = label
#                 b.confidence = float(score)
#                 b.x1, b.y1, b.x2, b.y2 = box
#                 if label == "person":
#                     has_person = True
#
#             if has_person:
#                 print("检测到人")
#
#             return response
#         except Exception as e:
#             print("推理异常:", str(e))
#             return analysis_pb2.InferResponse()
#
# # 启动服务
# def serve():
#     # 放大gRPC包大小
#     options = [
#         ('grpc.max_receive_message_length', 64 * 1024 * 1024),
#         ('grpc.max_send_message_length', 64 * 1024 * 1024),
#         # 减少gRPC内部缓冲延迟
#         ('grpc.http2.max_pings_without_data', 0),
#     ]
#     # 线程池
#     server = grpc.server(
#         futures.ThreadPoolExecutor(max_workers=10),
#         options=options
#     )
#
#     analysis_pb2_grpc.add_InferServiceServicer_to_server(InferService(), server)
#     server.add_insecure_port("127.0.0.1:50051")
#     print("TensorRT gRPC服务已启动：127.0.0.1:50051")
#     server.start()
#     server.wait_for_termination()
#
# if __name__ == "__main__":
#     serve()

import time
import cv2
import grpc
import numpy as np
import torch
import tensorrt as trt
from concurrent import futures
from threading import Lock

import analysis_pb2
import analysis_pb2_grpc

#全局配置
ENGINE_PATH = "best.engine"
INPUT_W = 640
INPUT_H = 640
CONF_THRESH = 0.65
NMS_THRESH = 0.75
CONTEXT_NUM = 18
TRT_LOGGER = trt.Logger(trt.Logger.ERROR)
YOLO_CLASSES = ["person", "car"]

# TensorRT多路CUDA流推理池
class TensorRTInfer:
    def __init__(self, engine_path):
        with open(engine_path, "rb") as f, trt.Runtime(TRT_LOGGER) as runtime:
            self.engine = runtime.deserialize_cuda_engine(f.read())

        self.contexts = []
        self.streams = []
        self.lock = Lock()

        # 预创建多套独立上下文+CUDA流，并发无锁抢占
        for _ in range(CONTEXT_NUM):
            ctx = self.engine.create_execution_context()
            stream = torch.cuda.Stream()
            self.contexts.append(ctx)
            self.streams.append(stream)

        self.input_name = self.engine.get_tensor_name(0)
        self.output_name = self.engine.get_tensor_name(1)

    def get_resource(self):
        with self.lock:
            ctx = self.contexts.pop()
            stream = self.streams.pop()
        return ctx, stream

    def release_resource(self, ctx, stream):
        with self.lock:
            self.contexts.append(ctx)
            self.streams.append(stream)

    def infer(self, input_array):
        ctx, stream = self.get_resource()
        try:
            with torch.cuda.stream(stream):
                inp = torch.from_numpy(input_array).cuda(non_blocking=True).contiguous()
                ctx.set_tensor_address(self.input_name, inp.data_ptr())

                out_shape = tuple(ctx.get_tensor_shape(self.output_name))
                out = torch.empty(out_shape, dtype=torch.float32, device='cuda').contiguous()
                ctx.set_tensor_address(self.output_name, out.data_ptr())

                ctx.execute_async_v3(stream.cuda_stream)
                stream.synchronize()
                return out.cpu().numpy()
        finally:
            self.release_resource(ctx, stream)

trt_model = TensorRTInfer(ENGINE_PATH)

#Letterbox 预处理
def letterbox(img, target_size=640):
    h, w = img.shape[:2]
    scale = min(target_size / w, target_size / h)
    nw, nh = int(w * scale), int(h * scale)
    img_resized = cv2.resize(img, (nw, nh), interpolation=cv2.INTER_LINEAR)

    # 填充灰色边框114，和YOLO训练预处理统一
    dw = target_size - nw
    dh = target_size - nh
    top, bottom = dh // 2, dh - dh // 2
    left, right = dw // 2, dw - dw // 2
    img_pad = cv2.copyMakeBorder(
        img_resized, top, bottom, left, right,
        cv2.BORDER_CONSTANT, value=(114, 114, 114)
    )
    return img_pad, scale, (dw, dh)

def preprocess(image_rgba):
    """
    入参：Go传递过来的原始分辨率RGBA图
    返回：模型NCHW输入张量、缩放比例、填充宽高偏移
    """
    # RGBA -> BGR
    img_bgr = cv2.cvtColor(image_rgba, cv2.COLOR_RGBA2BGR)
    # letterbox等比缩放+填充
    img_pad, scale, pad = letterbox(img_bgr, INPUT_W)
    # BGR转RGB，匹配YOLO训练输入通道顺序（解决颜色颠倒误报）
    img_rgb = cv2.cvtColor(img_pad, cv2.COLOR_BGR2RGB)
    # 归一化0~1
    img_rgb = img_rgb.astype(np.float32) / 255.0
    # HWC -> NCHW
    img_nchw = np.transpose(img_rgb, (2, 0, 1))
    img_batch = np.expand_dims(img_nchw, axis=0)
    return img_batch, scale, pad

# 后处理：移除letterbox偏移，还原原图真实坐标
def postprocess(output, orig_w, orig_h, scale, pad):
    dw_total, dh_total = pad
    half_dw = dw_total / 2.0
    half_dh = dh_total / 2.0

    output = output.squeeze(0).transpose(1, 0)
    boxes = output[:, :4]
    scores = output[:, 4:]
    max_scores = np.max(scores, axis=1)
    class_ids = np.argmax(scores, axis=1)

    # 置信过滤
    keep_mask = max_scores >= CONF_THRESH
    boxes = boxes[keep_mask]
    max_scores = max_scores[keep_mask]
    class_ids = class_ids[keep_mask]

    if len(boxes) == 0:
        return []

    # YOLO输出 cx, cy, w, h
    cx = boxes[:, 0]
    cy = boxes[:, 1]
    bw = boxes[:, 2]
    bh = boxes[:, 3]

    # 640尺寸下的xyxy
    x1_640 = cx - bw / 2.0
    y1_640 = cy - bh / 2.0
    x2_640 = cx + bw / 2.0
    y2_640 = cy + bh / 2.0

    # 减去填充灰边偏移
    x1_noscale = x1_640 - half_dw
    x2_noscale = x2_640 - half_dw
    y1_noscale = y1_640 - half_dh
    y2_noscale = y2_640 - half_dh

    # 缩放回原始摄像头分辨率
    x1 = x1_noscale / scale
    y1 = y1_noscale / scale
    x2 = x2_noscale / scale
    y2 = y2_noscale / scale

    # NMS输入格式 x,y,w,h
    nms_w = x2 - x1
    nms_h = y2 - y1
    nms_boxes = np.stack([x1, y1, nms_w, nms_h], axis=1)

    indices = cv2.dnn.NMSBoxes(
        nms_boxes.tolist(),
        max_scores.tolist(),
        CONF_THRESH,
        NMS_THRESH
    )

    res = []
    for idx in indices:
        i = idx[0] if isinstance(idx, (list, np.ndarray)) else idx
        box_xyxy = [float(x1[i]), float(y1[i]), float(x2[i]), float(y2[i])]
        res.append((box_xyxy, float(max_scores[i]), int(class_ids[i])))
    return res

#  gRPC推理服务实现
class InferService(analysis_pb2_grpc.InferServiceServicer):
    def Infer(self, request, context):
        try:
            # 解析Go传来的原始分辨率RGBA帧
            raw_data = np.frombuffer(request.image_data, np.uint8)
            orig_h = request.height
            orig_w = request.width
            img_rgba = raw_data.reshape((orig_h, orig_w, 4))

            # 预处理letterbox
            input_tensor, scale, pad = preprocess(img_rgba)
            # TRT推理
            trt_out = trt_model.infer(input_tensor)
            # 后处理还原原图坐标
            dets = postprocess(trt_out, orig_w, orig_h, scale, pad)

            resp = analysis_pb2.InferResponse()
            has_person = False

            for box, score, cid in dets:
                label = YOLO_CLASSES[cid]
                b_proto = resp.boxes.add()
                b_proto.label = label
                b_proto.confidence = score
                b_proto.x1, b_proto.y1, b_proto.x2, b_proto.y2 = box
                if label == "person":
                    has_person = True

            if has_person:
               print(f"[{time.strftime('%H:%M:%S')}] 检测到人")
            return resp

        except Exception as e:
            print(f"[{time.strftime('%H:%M:%S')}]推理异常:", str(e))
            import traceback
            traceback.print_exc()
            return analysis_pb2.InferResponse()

# 启动gRPC服
def serve():
    options = [
        ('grpc.max_receive_message_length', 64 * 1024 * 1024),  # 大图扩容到64MB
        ('grpc.max_send_message_length', 64 * 1024 * 1024),
        ('grpc.http2.max_pings_without_data', 0),
    ]
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=16),
        options=options
    )
    analysis_pb2_grpc.add_InferServiceServicer_to_server(InferService(), server)
    server.add_insecure_port("127.0.0.1:50051")
    print("TensorRT gRPC服务启动成功 127.0.0.1:50051")
    server.start()
    server.wait_for_termination()

if __name__ == "__main__":
    serve()