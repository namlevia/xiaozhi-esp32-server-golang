package controllers

import "strings"

// VoiceOption là tùy chọn giọng nói
type VoiceOption struct {
	Value           string  `json:"value"`                       // Giá trị giọng
	Label           string  `json:"label"`                       // Tên hiển thị giọng
	ModelPath       string  `json:"model_path,omitempty"`        // Đường dẫn model Piper
	ModelConfigPath string  `json:"model_config_path,omitempty"` // Đường dẫn metadata Piper
	SampleRate      int     `json:"sample_rate,omitempty"`       // Sample rate của giọng
	Language        string  `json:"language,omitempty"`          // Ngôn ngữ của giọng
	LengthScale     float32 `json:"length_scale,omitempty"`      // Piper length scale
	NoiseScale      float32 `json:"noise_scale,omitempty"`       // Piper noise scale
	NoiseW          float32 `json:"noise_w,omitempty"`           // Piper noise W
}

// VoiceOptions định nghĩa tùy chọn giọng của từng provider
// Theo tài liệu giọng nói Doubao của Volcengine.
// Và tài liệu Doubao WebSocket.
var VoiceOptions = map[string][]VoiceOption{
	// Danh sách giọng Edge TTS tiếng Trung
	// Tham khảo danh sách giọng Edge TTS thường dùng.
	"edge": {
		{Value: "zh-CN-XiaoxiaoNeural", Label: "Giọng zh-CN-XiaoxiaoNeural"},
		{Value: "zh-CN-YunxiNeural", Label: "Giọng zh-CN-YunxiNeural"},
		{Value: "zh-CN-YunyangNeural", Label: "Giọng zh-CN-YunyangNeural"},
		{Value: "zh-CN-XiaoyiNeural", Label: "Giọng zh-CN-XiaoyiNeural"},
		{Value: "zh-CN-YunjianNeural", Label: "Giọng zh-CN-YunjianNeural"},
		{Value: "zh-CN-YunxiaNeural", Label: "Giọng zh-CN-YunxiaNeural"},
		{Value: "zh-CN-YunhaoNeural", Label: "Giọng zh-CN-YunhaoNeural"},
		{Value: "zh-CN-XiaohanNeural", Label: "Giọng zh-CN-XiaohanNeural"},
		{Value: "zh-CN-XiaomoNeural", Label: "Giọng zh-CN-XiaomoNeural"},
		{Value: "zh-CN-XiaoxuanNeural", Label: "Giọng zh-CN-XiaoxuanNeural"},
		{Value: "zh-CN-XiaoruiNeural", Label: "Giọng zh-CN-XiaoruiNeural"},
		{Value: "zh-CN-XiaoshuangNeural", Label: "Giọng zh-CN-XiaoshuangNeural"},
		{Value: "zh-CN-XiaoyanNeural", Label: "Giọng zh-CN-XiaoyanNeural"},
		{Value: "zh-CN-XiaoyouNeural", Label: "Giọng zh-CN-XiaoyouNeural"},
		{Value: "zh-CN-XiaozhenNeural", Label: "Giọng zh-CN-XiaozhenNeural"},
		{Value: "zh-CN-YunfengNeural", Label: "Giọng zh-CN-YunfengNeural"},
		{Value: "zh-CN-YunyeNeural", Label: "Giọng zh-CN-YunyeNeural"},
		{Value: "zh-CN-YunzeNeural", Label: "Giọng zh-CN-YunzeNeural"},
	},

	// Danh sách giọng Microsoft TTS tiếng Trung
	"microsoft": {
		{Value: "zh-CN-XiaoxiaoNeural", Label: "Giọng zh-CN-XiaoxiaoNeural"},
		{Value: "zh-CN-YunxiNeural", Label: "Giọng zh-CN-YunxiNeural"},
		{Value: "zh-CN-YunyangNeural", Label: "Giọng zh-CN-YunyangNeural"},
		{Value: "zh-CN-XiaoyiNeural", Label: "Giọng zh-CN-XiaoyiNeural"},
		{Value: "zh-CN-YunjianNeural", Label: "Giọng zh-CN-YunjianNeural"},
		{Value: "zh-CN-YunxiaNeural", Label: "Giọng zh-CN-YunxiaNeural"},
		{Value: "zh-CN-YunhaoNeural", Label: "Giọng zh-CN-YunhaoNeural"},
		{Value: "zh-CN-XiaohanNeural", Label: "Giọng zh-CN-XiaohanNeural"},
		{Value: "zh-CN-XiaomoNeural", Label: "Giọng zh-CN-XiaomoNeural"},
		{Value: "zh-CN-XiaoxuanNeural", Label: "Giọng zh-CN-XiaoxuanNeural"},
		{Value: "zh-CN-XiaoruiNeural", Label: "Giọng zh-CN-XiaoruiNeural"},
		{Value: "zh-CN-XiaoshuangNeural", Label: "Giọng zh-CN-XiaoshuangNeural"},
		{Value: "zh-CN-XiaoyanNeural", Label: "Giọng zh-CN-XiaoyanNeural"},
		{Value: "zh-CN-XiaoyouNeural", Label: "Giọng zh-CN-XiaoyouNeural"},
		{Value: "zh-CN-XiaozhenNeural", Label: "Giọng zh-CN-XiaozhenNeural"},
		{Value: "zh-CN-YunfengNeural", Label: "Giọng zh-CN-YunfengNeural"},
		{Value: "zh-CN-YunyeNeural", Label: "Giọng zh-CN-YunyeNeural"},
		{Value: "zh-CN-YunzeNeural", Label: "Giọng zh-CN-YunzeNeural"},
	},

	// Danh sách giọng Doubao TTS qua HTTP
	// Tham khảo tài liệu Volcengine TTS.
	"doubao": {
		{Value: "BV700_V2_streaming", Label: "Giọng BV700_V2_streaming"},
		{Value: "BV705_streaming", Label: "Giọng BV705_streaming"},
		{Value: "BV701_V2_streaming", Label: "Giọng BV701_V2_streaming"},
		{Value: "BV001_V2_streaming", Label: "Giọng BV001_V2_streaming"},
		{Value: "BV700_streaming", Label: "Giọng BV700_streaming"},
		{Value: "BV406_V2_streaming", Label: "Giọng BV406_V2_streaming"},
		{Value: "BV406_streaming", Label: "Giọng BV406_streaming"},
		{Value: "BV407_V2_streaming", Label: "Giọng BV407_V2_streaming"},
		{Value: "BV407_streaming", Label: "Giọng BV407_streaming"},
		{Value: "BV001_streaming", Label: "Giọng BV001_streaming"},
		{Value: "BV002_streaming", Label: "Giọng BV002_streaming"},
		{Value: "BV701_streaming", Label: "Giọng BV701_streaming"},
		{Value: "BV119_streaming", Label: "Giọng BV119_streaming"},
		{Value: "BV102_streaming", Label: "Giọng BV102_streaming"},
		{Value: "BV113_streaming", Label: "Giọng BV113_streaming"},
		{Value: "BV115_streaming", Label: "Giọng BV115_streaming"},
		{Value: "BV007_streaming", Label: "Giọng BV007_streaming"},
		{Value: "BV056_streaming", Label: "Giọng BV056_streaming"},
		{Value: "BV005_streaming", Label: "Giọng BV005_streaming"},
		{Value: "BV051_streaming", Label: "Giọng BV051_streaming"},
		{Value: "BV034_streaming", Label: "Giọng BV034_streaming"},
		{Value: "BV033_streaming", Label: "Giọng BV033_streaming"},
		{Value: "BV021_streaming", Label: "Giọng BV021_streaming"},
		{Value: "BV019_streaming", Label: "Giọng BV019_streaming"},
		{Value: "BV213_streaming", Label: "Giọng BV213_streaming"},
		{Value: "BV503_streaming", Label: "Giọng BV503_streaming"},
		{Value: "BV504_streaming", Label: "Giọng BV504_streaming"},
		{Value: "BV522_streaming", Label: "Giọng BV522_streaming"},
		{Value: "BV524_streaming", Label: "Giọng BV524_streaming"},
		{Value: "BV104_streaming", Label: "Giọng BV104_streaming"},
		{Value: "BV004_streaming", Label: "Giọng BV004_streaming"},
		{Value: "BV009_streaming", Label: "Giọng BV009_streaming"},
		{Value: "BV008_streaming", Label: "Giọng BV008_streaming"},
		{Value: "BV064_streaming", Label: "Giọng BV064_streaming"},
		{Value: "BV437_streaming", Label: "Giọng BV437_streaming"},
		{Value: "BV511_streaming", Label: "Giọng BV511_streaming"},
		{Value: "BV040_streaming", Label: "Giọng BV040_streaming"},
		{Value: "BV138_streaming", Label: "Giọng BV138_streaming"},
		{Value: "BV704_streaming", Label: "Giọng BV704_streaming"},
		{Value: "BV702_streaming", Label: "Stefan"},
		{Value: "BV421_streaming", Label: "Giọng BV421_streaming"},
	},

	// Danh sách giọng Doubao WebSocket TTS
	// Tham khảo tài liệu chính thức về danh sách giọng:
	// https://www.volcengine.com/docs/6561/1257544?lang=zh
	// File này duy trì các giọng online chính thức thường dùng trong dự án.
	// Lưu ý: danh sách giọng chỉ dùng làm tùy chọn hiển thị, không ràng buộc cứng model/resource_id theo tên giọng.
	// Tính khả dụng thực tế phụ thuộc tài nguyên đã bật trong console Volcengine của appid/access_token hiện tại.

	"doubao_ws": {
		// Giọng nữ
		{Value: "zh_female_cancan_mars_bigtts", Label: "Giọng zh_female_cancan_mars_bigtts"},
		{Value: "zh_female_vv_uranus_bigtts", Label: "Giọng zh_female_vv_uranus_bigtts"},
		{Value: "zh_female_vv_jupiter_bigtts", Label: "Giọng zh_female_vv_jupiter_bigtts"},
		{Value: "zh_female_xiaohe_jupiter_bigtts", Label: "Giọng zh_female_xiaohe_jupiter_bigtts"},
		{Value: "saturn_zh_female_cancan_tob", Label: "Giọng saturn_zh_female_cancan_tob"},
		{Value: "saturn_zh_female_keainvsheng_tob", Label: "Giọng saturn_zh_female_keainvsheng_tob"},
		{Value: "saturn_zh_female_tiaopigongzhu_tob", Label: "Giọng saturn_zh_female_tiaopigongzhu_tob"},
		{Value: "zh_female_xiaohe_uranus_bigtts", Label: "Giọng zh_female_xiaohe_uranus_bigtts"},
		{Value: "zh_female_tianmeitaozi_mars_bigtts", Label: "Giọng zh_female_tianmeitaozi_mars_bigtts"},
		{Value: "zh_female_wanwanxiaohe_moon_bigtts", Label: "Giọng zh_female_wanwanxiaohe_moon_bigtts"},
		{Value: "zh_female_qinqienvsheng_moon_bigtts", Label: "Giọng zh_female_qinqienvsheng_moon_bigtts"},
		{Value: "zh_female_vv_mars_bigtts", Label: "Giọng zh_female_vv_mars_bigtts"},
		{Value: "zh_female_tianmeixiaoyuan_moon_bigtts", Label: "Giọng zh_female_tianmeixiaoyuan_moon_bigtts"},
		{Value: "zh_female_qingchezizi_moon_bigtts", Label: "Giọng zh_female_qingchezizi_moon_bigtts"},
		{Value: "zh_female_kailangjiejie_moon_bigtts", Label: "Giọng zh_female_kailangjiejie_moon_bigtts"},
		{Value: "zh_female_tianmeiyueyue_moon_bigtts", Label: "Giọng zh_female_tianmeiyueyue_moon_bigtts"},
		{Value: "zh_female_xinlingjitang_moon_bigtts", Label: "Giọng zh_female_xinlingjitang_moon_bigtts"},
		{Value: "zh_female_zhixingnvsheng_mars_bigtts", Label: "Giọng zh_female_zhixingnvsheng_mars_bigtts"},
		{Value: "zh_female_wenroushunv_mars_bigtts", Label: "Giọng zh_female_wenroushunv_mars_bigtts"},
		{Value: "zh_female_wenrouxiaoya_moon_bigtts", Label: "Giọng zh_female_wenrouxiaoya_moon_bigtts"},
		{Value: "zh_female_linjianvhai_moon_bigtts", Label: "Giọng zh_female_linjianvhai_moon_bigtts"},
		{Value: "zh_female_shuangkuaisisi_moon_bigtts", Label: "Giọng zh_female_shuangkuaisisi_moon_bigtts"},
		{Value: "zh_female_gaolengyujie_moon_bigtts", Label: "Giọng zh_female_gaolengyujie_moon_bigtts"},
		{Value: "zh_female_meilinvyou_moon_bigtts", Label: "Giọng zh_female_meilinvyou_moon_bigtts"},
		{Value: "zh_female_sajiaonvyou_moon_bigtts", Label: "Giọng zh_female_sajiaonvyou_moon_bigtts"},
		{Value: "zh_female_yuanqinvyou_moon_bigtts", Label: "Giọng zh_female_yuanqinvyou_moon_bigtts"},
		{Value: "ICL_zh_female_wenrounvshen_239eff5e8ffa_tob", Label: "Giọng ICL_zh_female_wenrounvshen_239eff5e8ffa_tob"},
		{Value: "ICL_zh_female_chunzhenshaonv_e588402fb8ad_tob", Label: "Giọng ICL_zh_female_chunzhenshaonv_e588402fb8ad_tob"},
		{Value: "ICL_zh_female_jinglingxiangdao_1beb294a9e3e_tob", Label: "Giọng ICL_zh_female_jinglingxiangdao_1beb294a9e3e_tob"},
		{Value: "ICL_zh_female_yilin_tob", Label: "Giọng ICL_zh_female_yilin_tob"},
		{Value: "ICL_zh_female_chengshujiejie_tob", Label: "Giọng ICL_zh_female_chengshujiejie_tob"},
		{Value: "ICL_zh_female_bingjiaojiejie_tob", Label: "Giọng ICL_zh_female_bingjiaojiejie_tob"},
		{Value: "ICL_zh_female_wumeiyujie_tob", Label: "Giọng ICL_zh_female_wumeiyujie_tob"},
		{Value: "ICL_zh_female_aojiaonvyou_tob", Label: "Giọng ICL_zh_female_aojiaonvyou_tob"},
		{Value: "ICL_zh_female_tiexinnvyou_tob", Label: "Giọng ICL_zh_female_tiexinnvyou_tob"},
		{Value: "ICL_zh_female_xingganyujie_tob", Label: "Giọng ICL_zh_female_xingganyujie_tob"},
		{Value: "ICL_zh_female_lixingyuanzi_cs_tob", Label: "Giọng ICL_zh_female_lixingyuanzi_cs_tob"},
		{Value: "ICL_zh_female_wuxi_tob", Label: "Giọng ICL_zh_female_wuxi_tob"},
		{Value: "ICL_zh_female_zhixingwenwan_tob", Label: "Giọng ICL_zh_female_zhixingwenwan_tob"},

		// Giọng nam
		{Value: "saturn_zh_male_shuanglangshaonian_tob", Label: "Giọng saturn_zh_male_shuanglangshaonian_tob"},
		{Value: "saturn_zh_male_tiancaitongzhuo_tob", Label: "Giọng saturn_zh_male_tiancaitongzhuo_tob"},
		{Value: "zh_male_yunzhou_jupiter_bigtts", Label: "Giọng zh_male_yunzhou_jupiter_bigtts"},
		{Value: "zh_male_xiaotian_jupiter_bigtts", Label: "Giọng zh_male_xiaotian_jupiter_bigtts"},
		{Value: "zh_male_m191_uranus_bigtts", Label: "Giọng zh_male_m191_uranus_bigtts"},
		{Value: "zh_male_taocheng_uranus_bigtts", Label: "Giọng zh_male_taocheng_uranus_bigtts"},
		{Value: "en_male_tim_uranus_bigtts", Label: "Giọng en_male_tim_uranus_bigtts"},
		{Value: "zh_male_yangguangqingnian_moon_bigtts", Label: "Giọng zh_male_yangguangqingnian_moon_bigtts"},
		{Value: "zh_male_qingshuangnanda_mars_bigtts", Label: "Giọng zh_male_qingshuangnanda_mars_bigtts"},
		{Value: "zh_male_wenrouxiaoge_mars_bigtts", Label: "Giọng zh_male_wenrouxiaoge_mars_bigtts"},
		{Value: "zh_male_qingcang_mars_bigtts", Label: "Giọng zh_male_qingcang_mars_bigtts"},
		{Value: "zh_male_ruyaqingnian_mars_bigtts", Label: "Giọng zh_male_ruyaqingnian_mars_bigtts"},
		{Value: "zh_male_jieshuoxiaoming_moon_bigtts", Label: "Giọng zh_male_jieshuoxiaoming_moon_bigtts"},
		{Value: "zh_male_linjiananhai_moon_bigtts", Label: "Giọng zh_male_linjiananhai_moon_bigtts"},
		{Value: "zh_male_yuanboxiaoshu_moon_bigtts", Label: "Giọng zh_male_yuanboxiaoshu_moon_bigtts"},
		{Value: "zh_male_wennuanahu_moon_bigtts", Label: "Giọng zh_male_wennuanahu_moon_bigtts"},
		{Value: "zh_male_shaonianzixin_moon_bigtts", Label: "Giọng zh_male_shaonianzixin_moon_bigtts"},
		{Value: "zh_male_beijingxiaoye_moon_bigtts", Label: "Giọng zh_male_beijingxiaoye_moon_bigtts"},
		{Value: "zh_male_jingqiangkanye_moon_bigtts", Label: "Giọng zh_male_jingqiangkanye_moon_bigtts"},
		{Value: "zh_male_guozhoudege_moon_bigtts", Label: "Giọng zh_male_guozhoudege_moon_bigtts"},
		{Value: "zh_male_haoyuxiaoge_moon_bigtts", Label: "Giọng zh_male_haoyuxiaoge_moon_bigtts"},
		{Value: "zh_male_shenyeboke_moon_bigtts", Label: "Giọng zh_male_shenyeboke_moon_bigtts"},
		{Value: "zh_male_aojiaobazong_moon_bigtts", Label: "Giọng zh_male_aojiaobazong_moon_bigtts"},
		{Value: "zh_male_dongfanghaoran_moon_bigtts", Label: "Giọng zh_male_dongfanghaoran_moon_bigtts"},
		{Value: "zh_male_M100_conversation_wvae_bigtts", Label: "Giọng zh_male_M100_conversation_wvae_bigtts"},
		{Value: "zh_male_xudong_conversation_wvae_bigtts", Label: "Giọng zh_male_xudong_conversation_wvae_bigtts"},
		{Value: "zh_male_qingyiyuxuan_mars_bigtts", Label: "Giọng zh_male_qingyiyuxuan_mars_bigtts"},
		{Value: "en_male_jason_conversation_wvae_bigtts", Label: "Giọng en_male_jason_conversation_wvae_bigtts"},
		{Value: "ICL_zh_male_lengkugege_v1_tob", Label: "Giọng ICL_zh_male_lengkugege_v1_tob"},
		{Value: "ICL_zh_male_shenmi_v1_tob", Label: "Giọng ICL_zh_male_shenmi_v1_tob"},
		{Value: "ICL_zh_male_BV705_streaming_cs_tob", Label: "Giọng ICL_zh_male_BV705_streaming_cs_tob"},
		{Value: "ICL_zh_male_menyoupingxiaoge_ffed9fc2fee7_tob", Label: "Giọng ICL_zh_male_menyoupingxiaoge_ffed9fc2fee7_tob"},
		{Value: "ICL_zh_male_anrenqinzhu_cd62e63dcdab_tob", Label: "Giọng ICL_zh_male_anrenqinzhu_cd62e63dcdab_tob"},
		{Value: "ICL_zh_male_guaogongzi_v1_tob", Label: "Giọng ICL_zh_male_guaogongzi_v1_tob"},
		{Value: "ICL_zh_male_bingruogongzi_tob", Label: "Giọng ICL_zh_male_bingruogongzi_tob"},
		{Value: "ICL_zh_male_bingjiaodidi_tob", Label: "Giọng ICL_zh_male_bingjiaodidi_tob"},
		{Value: "ICL_zh_male_aomanshaoye_tob", Label: "Giọng ICL_zh_male_aomanshaoye_tob"},
		{Value: "ICL_zh_male_chunzhenxuedi_tob", Label: "Giọng ICL_zh_male_chunzhenxuedi_tob"},
		{Value: "ICL_zh_male_yourougongzi_tob", Label: "Giọng ICL_zh_male_yourougongzi_tob"},
		{Value: "ICL_zh_male_tiexinnanyou_tob", Label: "Giọng ICL_zh_male_tiexinnanyou_tob"},
		{Value: "ICL_zh_male_shaonianjiangjun_tob", Label: "Giọng ICL_zh_male_shaonianjiangjun_tob"},
		{Value: "ICL_zh_male_bingjiaogege_tob", Label: "Giọng ICL_zh_male_bingjiaogege_tob"},
		{Value: "ICL_zh_male_xuebanantongzhuo_tob", Label: "Giọng ICL_zh_male_xuebanantongzhuo_tob"},
		{Value: "ICL_zh_male_youmoshushu_tob", Label: "Giọng ICL_zh_male_youmoshushu_tob"},
		{Value: "ICL_zh_male_wenrounantongzhuo_tob", Label: "Giọng ICL_zh_male_wenrounantongzhuo_tob"},
		{Value: "ICL_zh_male_youmodaye_tob", Label: "Giọng ICL_zh_male_youmodaye_tob"},
		{Value: "ICL_zh_male_shenmifashi_tob", Label: "Giọng ICL_zh_male_shenmifashi_tob"},
		{Value: "ICL_zh_male_lengjunshangsi_tob", Label: "Giọng ICL_zh_male_lengjunshangsi_tob"},
		{Value: "ICL_en_male_michael_tob", Label: "Giọng ICL_en_male_michael_tob"},

		// Giọng IP/đặc sắc
		{Value: "zh_male_lubanqihao_mars_bigtts", Label: "Giọng zh_male_lubanqihao_mars_bigtts"},
		{Value: "zh_female_yangmi_mars_bigtts", Label: "Giọng zh_female_yangmi_mars_bigtts"},
		{Value: "zh_female_linzhiling_mars_bigtts", Label: "Giọng zh_female_linzhiling_mars_bigtts"},
		{Value: "zh_female_jiyejizi2_mars_bigtts", Label: "Giọng zh_female_jiyejizi2_mars_bigtts"},
		{Value: "zh_male_tangseng_mars_bigtts", Label: "Giọng zh_male_tangseng_mars_bigtts"},
		{Value: "zh_male_zhubajie_mars_bigtts", Label: "Giọng zh_male_zhubajie_mars_bigtts"},
		{Value: "zh_female_naying_mars_bigtts", Label: "Giọng zh_female_naying_mars_bigtts"},
		{Value: "zh_female_leidian_mars_bigtts", Label: "Giọng zh_female_leidian_mars_bigtts"},
		{Value: "zh_male_sunwukong_mars_bigtts", Label: "Giọng zh_male_sunwukong_mars_bigtts"},
		{Value: "zh_male_xionger_mars_bigtts", Label: "Giọng zh_male_xionger_mars_bigtts"},
		{Value: "zh_female_peiqi_mars_bigtts", Label: "Giọng zh_female_peiqi_mars_bigtts"},
		{Value: "zh_female_yingtaowanzi_mars_bigtts", Label: "Giọng zh_female_yingtaowanzi_mars_bigtts"},
		{Value: "zh_male_silang_mars_bigtts", Label: "Giọng zh_male_silang_mars_bigtts"},
	},

	// Danh sách giọng Minimax TTS
	// Tham khảo tài liệu Minimax TTS.
	"minimax": {
		// Tiếng Trung phổ thông
		{Value: "male-qn-qingse", Label: "Giọng male-qn-qingse"},
		{Value: "male-qn-jingying", Label: "Giọng male-qn-jingying"},
		{Value: "male-qn-badao", Label: "Giọng male-qn-badao"},
		{Value: "male-qn-daxuesheng", Label: "Giọng male-qn-daxuesheng"},
		{Value: "female-shaonv", Label: "Giọng female-shaonv"},
		{Value: "female-yujie", Label: "Giọng female-yujie"},
		{Value: "female-chengshu", Label: "Giọng female-chengshu"},
		{Value: "female-tianmei", Label: "Giọng female-tianmei"},
		{Value: "male-qn-qingse-jingpin", Label: "Giọng male-qn-qingse-jingpin"},
		{Value: "male-qn-jingying-jingpin", Label: "Giọng male-qn-jingying-jingpin"},
		{Value: "male-qn-badao-jingpin", Label: "Giọng male-qn-badao-jingpin"},
		{Value: "male-qn-daxuesheng-jingpin", Label: "Giọng male-qn-daxuesheng-jingpin"},
		{Value: "female-shaonv-jingpin", Label: "Giọng female-shaonv-jingpin"},
		{Value: "female-yujie-jingpin", Label: "Giọng female-yujie-jingpin"},
		{Value: "female-chengshu-jingpin", Label: "Giọng female-chengshu-jingpin"},
		{Value: "female-tianmei-jingpin", Label: "Giọng female-tianmei-jingpin"},
		{Value: "clever_boy", Label: "Giọng clever_boy"},
		{Value: "cute_boy", Label: "Giọng cute_boy"},
		{Value: "lovely_girl", Label: "Giọng lovely_girl"},
		{Value: "cartoon_pig", Label: "Giọng cartoon_pig"},
		{Value: "bingjiao_didi", Label: "Giọng bingjiao_didi"},
		{Value: "junlang_nanyou", Label: "Giọng junlang_nanyou"},
		{Value: "chunzhen_xuedi", Label: "Giọng chunzhen_xuedi"},
		{Value: "lengdan_xiongzhang", Label: "Giọng lengdan_xiongzhang"},
		{Value: "badao_shaoye", Label: "Giọng badao_shaoye"},
		{Value: "tianxin_xiaoling", Label: "Giọng tianxin_xiaoling"},
		{Value: "qiaopi_mengmei", Label: "Giọng qiaopi_mengmei"},
		{Value: "wumei_yujie", Label: "Giọng wumei_yujie"},
		{Value: "diadia_xuemei", Label: "Giọng diadia_xuemei"},
		{Value: "danya_xuejie", Label: "Giọng danya_xuejie"},
		{Value: "Chinese (Mandarin)_Reliable_Executive", Label: "Giọng Chinese (Mandarin)_Reliable_Executive"},
		{Value: "Chinese (Mandarin)_News_Anchor", Label: "Giọng Chinese (Mandarin)_News_Anchor"},
		{Value: "Chinese (Mandarin)_Mature_Woman", Label: "Giọng Chinese (Mandarin)_Mature_Woman"},
		{Value: "Chinese (Mandarin)_Unrestrained_Young_Man", Label: "Giọng Chinese (Mandarin)_Unrestrained_Young_Man"},
		{Value: "Arrogant_Miss", Label: "Giọng Arrogant_Miss"},
		{Value: "Robot_Armor", Label: "Giọng Robot_Armor"},
		{Value: "Chinese (Mandarin)_Kind-hearted_Antie", Label: "Giọng Chinese (Mandarin)_Kind-hearted_Antie"},
		{Value: "Chinese (Mandarin)_HK_Flight_Attendant", Label: "Giọng Chinese (Mandarin)_HK_Flight_Attendant"},
		{Value: "Chinese (Mandarin)_Humorous_Elder", Label: "Giọng Chinese (Mandarin)_Humorous_Elder"},
		{Value: "Chinese (Mandarin)_Gentleman", Label: "Giọng Chinese (Mandarin)_Gentleman"},
		{Value: "Chinese (Mandarin)_Warm_Bestie", Label: "Giọng Chinese (Mandarin)_Warm_Bestie"},
		{Value: "Chinese (Mandarin)_Male_Announcer", Label: "Giọng Chinese (Mandarin)_Male_Announcer"},
		{Value: "Chinese (Mandarin)_Sweet_Lady", Label: "Giọng Chinese (Mandarin)_Sweet_Lady"},
		{Value: "Chinese (Mandarin)_Southern_Young_Man", Label: "Giọng Chinese (Mandarin)_Southern_Young_Man"},
		{Value: "Chinese (Mandarin)_Wise_Women", Label: "Giọng Chinese (Mandarin)_Wise_Women"},
		{Value: "Chinese (Mandarin)_Gentle_Youth", Label: "Giọng Chinese (Mandarin)_Gentle_Youth"},
		{Value: "Chinese (Mandarin)_Warm_Girl", Label: "Giọng Chinese (Mandarin)_Warm_Girl"},
		{Value: "Chinese (Mandarin)_Kind-hearted_Elder", Label: "Giọng Chinese (Mandarin)_Kind-hearted_Elder"},
		{Value: "Chinese (Mandarin)_Cute_Spirit", Label: "Giọng Chinese (Mandarin)_Cute_Spirit"},
		{Value: "Chinese (Mandarin)_Radio_Host", Label: "Giọng Chinese (Mandarin)_Radio_Host"},
		{Value: "Chinese (Mandarin)_Lyrical_Voice", Label: "Giọng Chinese (Mandarin)_Lyrical_Voice"},
		{Value: "Chinese (Mandarin)_Straightforward_Boy", Label: "Giọng Chinese (Mandarin)_Straightforward_Boy"},
		{Value: "Chinese (Mandarin)_Sincere_Adult", Label: "Giọng Chinese (Mandarin)_Sincere_Adult"},
		{Value: "Chinese (Mandarin)_Gentle_Senior", Label: "Giọng Chinese (Mandarin)_Gentle_Senior"},
		{Value: "Chinese (Mandarin)_Stubborn_Friend", Label: "Giọng Chinese (Mandarin)_Stubborn_Friend"},
		{Value: "Chinese (Mandarin)_Crisp_Girl", Label: "Giọng Chinese (Mandarin)_Crisp_Girl"},
		{Value: "Chinese (Mandarin)_Pure-hearted_Boy", Label: "Giọng Chinese (Mandarin)_Pure-hearted_Boy"},
		{Value: "Chinese (Mandarin)_Soft_Girl", Label: "Giọng Chinese (Mandarin)_Soft_Girl"},
		// Tiếng Quảng Đông
		{Value: "Cantonese_ProfessionalHost（F)", Label: "Giọng Cantonese_ProfessionalHost（F)"},
		{Value: "Cantonese_GentleLady", Label: "Giọng Cantonese_GentleLady"},
		{Value: "Cantonese_ProfessionalHost（M)", Label: "Giọng Cantonese_ProfessionalHost（M)"},
		{Value: "Cantonese_PlayfulMan", Label: "Giọng Cantonese_PlayfulMan"},
		{Value: "Cantonese_CuteGirl", Label: "Giọng Cantonese_CuteGirl"},
		{Value: "Cantonese_KindWoman", Label: "Giọng Cantonese_KindWoman"},
		// Tiếng Anh
		{Value: "Santa_Claus", Label: "Santa Claus"},
		{Value: "Grinch", Label: "Grinch"},
		{Value: "Rudolph", Label: "Rudolph"},
		{Value: "Arnold", Label: "Arnold"},
		{Value: "Charming_Santa", Label: "Charming Santa"},
		{Value: "Charming_Lady", Label: "Charming Lady"},
		{Value: "Sweet_Girl", Label: "Sweet Girl"},
		{Value: "Cute_Elf", Label: "Cute Elf"},
		{Value: "Attractive_Girl", Label: "Attractive Girl"},
		{Value: "Serene_Woman", Label: "Serene Woman"},
		{Value: "English_Trustworthy_Man", Label: "Trustworthy Man"},
		{Value: "English_Graceful_Lady", Label: "Graceful Lady"},
		{Value: "English_Aussie_Bloke", Label: "Aussie Bloke"},
		{Value: "English_Whispering_girl", Label: "Whispering girl"},
		{Value: "English_Diligent_Man", Label: "Diligent Man"},
		{Value: "English_Gentle-voiced_man", Label: "Gentle-voiced man"},
	},

	// Danh sách giọng Aliyun Qwen TTS cơ bản, lọc theo model trong GetAliyunQwenVoicesByModel.
	"aliyun_qwen": {
		{Value: "Cherry", Label: "Giọng Cherry"},
		{Value: "Serena", Label: "Giọng Serena"},
		{Value: "Ethan", Label: "Giọng Ethan"},
		{Value: "Chelsie", Label: "Giọng Chelsie"},
		{Value: "Momo", Label: "Giọng Momo"},
		{Value: "Vivian", Label: "Giọng Vivian"},
		{Value: "Moon", Label: "Giọng Moon"},
		{Value: "Maia", Label: "Giọng Maia"},
		{Value: "Kai", Label: "Giọng Kai"},
		{Value: "Nofish", Label: "Giọng Nofish"},
		{Value: "Bella", Label: "Giọng Bella"},
		{Value: "Jennifer", Label: "Giọng Jennifer"},
		{Value: "Ryan", Label: "Giọng Ryan"},
	},

	// Danh sách giọng Xunfei Online TTS
	// Ghi chú: giữ một nhóm giọng tĩnh thường dùng, khả dụng thực tế phụ thuộc quyền trong console Xunfei.
	// Tham khảo:
	// https://www.xfyun.cn/doc/tts/online_tts/API.html
	// https://aiui.xfyun.cn/doc/aiui/3_access_service/access_interact/functions/speech_synthesis.html
	"xunfei": {
		{Value: "xiaoyan", Label: "Giọng xiaoyan"},
		{Value: "xiaofeng", Label: "Giọng xiaofeng"},
		{Value: "yezi", Label: "Giọng yezi"},
		{Value: "yifei", Label: "Giọng yifei"},
		{Value: "yiping", Label: "Giọng yiping"},
		{Value: "qige", Label: "Giọng qige"},
		{Value: "chaoge", Label: "Giọng chaoge"},
		{Value: "pengfei", Label: "Giọng pengfei"},
		{Value: "xiaoxin", Label: "Giọng xiaoxin"},
		{Value: "john", Label: "Giọng john"},
		{Value: "catherine", Label: "Giọng catherine"},
	},

	// Danh sách giọng Xunfei Super TTS
	// Ghi chú: giữ một nhóm giọng tĩnh đề xuất, khả dụng thực tế phụ thuộc quyền trong console Xunfei.
	"xunfei_super_tts": {
		{Value: "x6_lingxiaoxue_pro", Label: "Giọng x6_lingxiaoxue_pro"},
		{Value: "x6_lingfeiyi_pro", Label: "Giọng x6_lingfeiyi_pro"},
		{Value: "x6_lingxiaoli_pro", Label: "Giọng x6_lingxiaoli_pro"},
		{Value: "x6_lingxiaoyue_pro", Label: "Giọng x6_lingxiaoyue_pro"},
		{Value: "x6_lingxiaoxuan_pro", Label: "Giọng x6_lingxiaoxuan_pro"},
		{Value: "x6_lingyuyan_pro", Label: "Giọng x6_lingyuyan_pro"},
		{Value: "x6_lingyouyou_pro", Label: "Giọng x6_lingyouyou_pro"},
		{Value: "x6_feizheChat_pro", Label: "Giọng x6_feizheChat_pro"},
		{Value: "x6_xiaoqiChat_pro", Label: "Giọng x6_xiaoqiChat_pro"},
		{Value: "x5_lingxiaotang_flow", Label: "Giọng x5_lingxiaotang_flow"},
		{Value: "x5_lingyuzhao_flow", Label: "Giọng x5_lingyuzhao_flow"},
		{Value: "x4_zijin_oral", Label: "Giọng x4_zijin_oral"},
		{Value: "x4_ziyang_oral", Label: "Giọng x4_ziyang_oral"},
	},

	// Danh sách giọng Zhipu TTS
	"zhipu": {
		{Value: "tongtong", Label: "Giọng tongtong"},
		{Value: "chuichui", Label: "Giọng chuichui"},
		{Value: "xiaochen", Label: "Giọng xiaochen"},
		{Value: "jam", Label: "Giọng jam"},
		{Value: "kazi", Label: "Giọng kazi"},
		{Value: "douji", Label: "Giọng douji"},
		{Value: "luodo", Label: "Giọng luodo"},
	},
}

func normalizeVoiceLabel(label string) string {
	replacer := strings.NewReplacer(
		"(female)", " (giọng nữ)",
		"(male)", " (giọng nam)",
		"(default recommended)", " (mặc định đề xuất)",
		"(default voice)", " (giọng mặc định)",
		"(English male voice)", " (giọng nam tiếng Anh)",
		"(English female voice)", " (giọng nữ tiếng Anh)",
		"(American English male voice)", " (giọng nam tiếng Anh Mỹ)",
		"(customer service female voice)", " (giọng nữ CSKH)",
		"(x4, conversational)", " (x4, khẩu ngữ)",
		"（x6）", " (x6)",
		"（x5）", " (x5)",
		"O version", "bản O",
		"voice", "giọng",
		"female voice", "giọng nữ",
		"male voice", "giọng nam",
		"bilingual", "song ngữ",
		"multi-emotion", "đa cảm xúc",
	)
	return replacer.Replace(label)
}

func normalizeVoiceOptions(options []VoiceOption) []VoiceOption {
	result := make([]VoiceOption, 0, len(options))
	for _, voice := range options {
		result = append(result, VoiceOption{
			Value: voice.Value,
			Label: normalizeVoiceLabel(voice.Label),
		})
	}
	return result
}

// GetVoiceOptionsByProvider lấy danh sách giọng theo provider
func GetVoiceOptionsByProvider(provider string) []VoiceOption {
	if voices, ok := VoiceOptions[provider]; ok {
		return normalizeVoiceOptions(voices)
	}
	return []VoiceOption{}
}

// GetAliyunQwenVoicesByModel lấy danh sách giọng theo tên model Qwen
// Dùng ánh xạ model Qwen để lấy danh sách giọng chính xác
func GetAliyunQwenVoicesByModel(model string) []VoiceOption {
	model = strings.TrimSpace(model)
	if model == "" {
		// Nếu không có model, trả về danh sách cơ bản
		return GetVoiceOptionsByProvider("aliyun_qwen")
	}

	// Dùng hàm cục bộ để lấy danh sách giọng tương ứng với model
	voices := GetVoicesByModel(model)
	if voices == nil || len(voices) == 0 {
		// Nếu không tìm thấy giọng cho model, trả về danh sách cơ bản
		return GetVoiceOptionsByProvider("aliyun_qwen")
	}

	// Chuyển VoiceInfo thành VoiceOption
	result := make([]VoiceOption, 0, len(voices))
	for _, v := range voices {
		result = append(result, VoiceOption{
			Value: v.Value,
			Label: v.Label,
		})
	}
	return normalizeVoiceOptions(result)
}
