@echo off
cd /d "%~dp0"
if not exist logs mkdir logs

echo Dang khoi dong Xiaozhi Server, log se ghi vao logs\startup.log
echo ==== %date% %time% ====>> logs\startup.log
xiaozhi_server.exe -c main_config.yaml -asr-enable --asr-config asr_server.json --manager-enable --manager-config manager.json -tts-enable -tts-config tts_server.json >> logs\startup.log 2>&1

echo Xiaozhi Server da thoat voi ma loi %errorlevel%. Xem logs\startup.log de biet chi tiet.
pause
