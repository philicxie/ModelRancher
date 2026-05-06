"""
MNIST 手写数字识别 CNN 训练脚本
兼容 CPU / GPU（CUDA / MPS）
通过环境变量读取数据路径和输出路径
"""

import os
import sys
import time
import json

import torch
import torch.nn as nn
import torch.optim as optim
import torch.nn.functional as F
from torch.utils.data import DataLoader
from torchvision import datasets, transforms

# ------------------------------------------------------------------
# 配置：通过环境变量读取，提供合理的默认值
# ------------------------------------------------------------------
DATA_PATH = os.environ.get("DATA_PATH", "/Users/phil/Desktop/lab/data")
OUTPUT_PATH = os.environ.get("OUTPUT_PATH", "/Users/phil/Desktop/lab/output")
TASK_ID = os.environ.get("TASK_ID", "local")

# 训练超参（也可通过环境变量覆盖）
BATCH_SIZE = int(os.environ.get("BATCH_SIZE", "128"))
EPOCHS = int(os.environ.get("EPOCHS", "10"))
LR = float(os.environ.get("LR", "0.001"))

# ------------------------------------------------------------------
# 自动选择设备
# ------------------------------------------------------------------
def get_device():
    if torch.cuda.is_available():
        device = torch.device("cuda")
        print(f"[Device] Using CUDA: {torch.cuda.get_device_name(0)}")
    elif torch.backends.mps.is_available():
        device = torch.device("mps")
        print("[Device] Using Apple MPS")
    else:
        device = torch.device("cpu")
        print("[Device] Using CPU")
    return device


# ------------------------------------------------------------------
# CNN 网络定义
# ------------------------------------------------------------------
class MNISTNet(nn.Module):
    def __init__(self):
        super(MNISTNet, self).__init__()
        # 卷积层：1x28x28 -> 32x14x14 -> 64x7x7
        self.conv1 = nn.Conv2d(1, 32, kernel_size=3, padding=1)
        self.conv2 = nn.Conv2d(32, 64, kernel_size=3, padding=1)
        self.pool = nn.MaxPool2d(2, 2)
        self.dropout1 = nn.Dropout2d(0.25)
        self.dropout2 = nn.Dropout(0.5)

        # 全连接层：64*7*7=3136 -> 128 -> 10
        self.fc1 = nn.Linear(64 * 7 * 7, 128)
        self.fc2 = nn.Linear(128, 10)

    def forward(self, x):
        x = self.pool(F.relu(self.conv1(x)))      # -> 32x14x14
        x = self.pool(F.relu(self.conv2(x)))      # -> 64x7x7
        x = self.dropout1(x)
        x = torch.flatten(x, 1)                   # -> 3136
        x = F.relu(self.fc1(x))                   # -> 128
        x = self.dropout2(x)
        x = self.fc2(x)                           # -> 10
        return F.log_softmax(x, dim=1)


# ------------------------------------------------------------------
# 数据加载
# ------------------------------------------------------------------
def load_data(data_dir, batch_size):
    transform = transforms.Compose([
        transforms.ToTensor(),
        transforms.Normalize((0.1307,), (0.3081,))  # MNIST 均值/方差
    ])

    train_dataset = datasets.MNIST(
        root=data_dir, train=True, download=False, transform=transform
    )
    test_dataset = datasets.MNIST(
        root=data_dir, train=False, download=False, transform=transform
    )

    train_loader = DataLoader(train_dataset, batch_size=batch_size, shuffle=True, num_workers=2)
    test_loader = DataLoader(test_dataset, batch_size=batch_size, shuffle=False, num_workers=2)

    return train_loader, test_loader


# ------------------------------------------------------------------
# 训练一个 epoch
# ------------------------------------------------------------------
def train_epoch(model, device, train_loader, optimizer, epoch):
    model.train()
    total_loss = 0.0
    correct = 0
    total = 0

    for batch_idx, (data, target) in enumerate(train_loader):
        data, target = data.to(device), target.to(device)

        optimizer.zero_grad()
        output = model(data)
        loss = F.nll_loss(output, target)
        loss.backward()
        optimizer.step()

        total_loss += loss.item()
        pred = output.argmax(dim=1)
        correct += pred.eq(target).sum().item()
        total += target.size(0)

        if batch_idx % 100 == 0:
            print(f"  Batch {batch_idx:4d}/{len(train_loader)}  Loss: {loss.item():.4f}")

    avg_loss = total_loss / len(train_loader)
    acc = 100.0 * correct / total
    print(f"[Epoch {epoch}] Train Loss: {avg_loss:.4f}  Acc: {acc:.2f}%")
    return avg_loss, acc


# ------------------------------------------------------------------
# 验证
# ------------------------------------------------------------------
def evaluate(model, device, test_loader):
    model.eval()
    test_loss = 0.0
    correct = 0

    with torch.no_grad():
        for data, target in test_loader:
            data, target = data.to(device), target.to(device)
            output = model(data)
            test_loss += F.nll_loss(output, target, reduction="sum").item()
            pred = output.argmax(dim=1)
            correct += pred.eq(target).sum().item()

    test_loss /= len(test_loader.dataset)
    acc = 100.0 * correct / len(test_loader.dataset)
    print(f"[Test] Loss: {test_loss:.4f}  Acc: {acc:.2f}% ({correct}/{len(test_loader.dataset)})")
    return test_loss, acc


# ------------------------------------------------------------------
# 主函数
# ------------------------------------------------------------------
def main():
    print("=" * 60)
    print("MNIST CNN Training")
    print(f"  TASK_ID:    {TASK_ID}")
    print(f"  DATA_PATH:  {DATA_PATH}")
    print(f"  OUTPUT_PATH:{OUTPUT_PATH}")
    print(f"  BATCH_SIZE: {BATCH_SIZE}")
    print(f"  EPOCHS:     {EPOCHS}")
    print(f"  LR:         {LR}")
    print("=" * 60)

    os.makedirs(DATA_PATH, exist_ok=True)
    os.makedirs(OUTPUT_PATH, exist_ok=True)

    device = get_device()
    train_loader, test_loader = load_data(DATA_PATH, BATCH_SIZE)

    model = MNISTNet().to(device)
    optimizer = optim.Adam(model.parameters(), lr=LR)
    scheduler = optim.lr_scheduler.StepLR(optimizer, step_size=5, gamma=0.5)

    history = []
    best_acc = 0.0

    for epoch in range(1, EPOCHS + 1):
        start = time.time()
        train_loss, train_acc = train_epoch(model, device, train_loader, optimizer, epoch)
        test_loss, test_acc = evaluate(model, device, test_loader)
        scheduler.step()
        elapsed = time.time() - start

        history.append({
            "epoch": epoch,
            "train_loss": train_loss,
            "train_acc": train_acc,
            "test_loss": test_loss,
            "test_acc": test_acc,
            "time": elapsed,
        })

        print(f"  Epoch {epoch} finished in {elapsed:.1f}s\n")

        # 保存最佳模型
        if test_acc > best_acc:
            best_acc = test_acc
            best_path = os.path.join(OUTPUT_PATH, "mnist_best.pt")
            torch.save({
                "epoch": epoch,
                "model_state_dict": model.state_dict(),
                "optimizer_state_dict": optimizer.state_dict(),
                "test_acc": test_acc,
            }, best_path)
            print(f"  -> Saved best model to {best_path} (acc={test_acc:.2f}%)")

    # 保存最终模型
    final_path = os.path.join(OUTPUT_PATH, "mnist_final.pt")
    torch.save(model.state_dict(), final_path)
    print(f"\nSaved final model to {final_path}")

    # 保存训练日志
    log_path = os.path.join(OUTPUT_PATH, "train_log.json")
    with open(log_path, "w") as f:
        json.dump({
            "task_id": TASK_ID,
            "device": str(device),
            "epochs": EPOCHS,
            "batch_size": BATCH_SIZE,
            "lr": LR,
            "best_acc": best_acc,
            "history": history,
        }, f, indent=2)
    print(f"Saved training log to {log_path}")
    print("Training complete!")


if __name__ == "__main__":
    main()
