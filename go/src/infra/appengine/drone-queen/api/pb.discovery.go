// Code generated by cproto. DO NOT EDIT.

package api

import discovery "go.chromium.org/luci/grpc/discovery"

import "github.com/golang/protobuf/protoc-gen-go/descriptor"

func init() {
	discovery.RegisterDescriptorSetCompressed(
		[]string{
			"drone_queen.Drone", "drone_queen.InventoryProvider", "drone_queen.Inspect",
		},
		[]byte{31, 139,
			8, 0, 0, 0, 0, 0, 0, 255, 164, 90, 77, 112, 27, 71,
			118, 198, 252, 144, 4, 155, 250, 99, 203, 150, 100, 72, 150, 158,
			32, 115, 13, 122, 193, 1, 127, 44, 57, 146, 236, 117, 64, 0,
			162, 32, 147, 0, 131, 31, 201, 146, 202, 43, 14, 48, 13, 96,
			164, 193, 12, 118, 186, 65, 154, 118, 28, 39, 135, 77, 213, 86,
			165, 114, 200, 33, 149, 67, 14, 249, 169, 218, 195, 230, 144, 75,
			246, 144, 202, 41, 247, 156, 115, 77, 229, 144, 107, 42, 199, 28,
			83, 175, 123, 102, 0, 144, 210, 106, 189, 97, 89, 229, 121, 61,
			221, 239, 231, 123, 175, 223, 123, 221, 3, 242, 143, 5, 114, 163,
			31, 4, 125, 143, 21, 70, 97, 32, 130, 206, 184, 87, 16, 238,
			144, 113, 97, 15, 71, 150, 28, 162, 231, 213, 4, 43, 158, 144,
			189, 79, 22, 91, 241, 28, 122, 133, 44, 112, 214, 13, 124, 135,
			95, 209, 64, 203, 25, 141, 152, 164, 239, 144, 57, 223, 246, 3,
			126, 69, 7, 45, 55, 215, 80, 196, 246, 31, 145, 139, 221, 96,
			104, 157, 224, 185, 125, 46, 225, 184, 143, 67, 251, 218, 179, 31,
			247, 93, 49, 24, 119, 172, 110, 48, 44, 244, 3, 207, 246, 251,
			19, 21, 71, 226, 120, 196, 248, 68, 211, 255, 213, 180, 191, 211,
			141, 157, 253, 237, 95, 233, 215, 119, 20, 231, 253, 104, 174, 245,
			132, 121, 222, 23, 126, 112, 228, 183, 112, 205, 163, 95, 175, 145,
			5, 58, 119, 61, 245, 167, 154, 70, 254, 237, 12, 209, 206, 80,
			227, 122, 138, 110, 254, 235, 25, 144, 43, 186, 129, 7, 219, 227,
			94, 143, 133, 28, 214, 64, 241, 250, 144, 131, 99, 11, 27, 92,
			95, 176, 176, 59, 176, 253, 62, 131, 94, 16, 14, 109, 65, 160,
			20, 140, 142, 67, 183, 63, 16, 176, 185, 190, 254, 123, 209, 2,
			168, 250, 93, 11, 160, 232, 121, 32, 223, 113, 8, 25, 103, 225,
			33, 115, 44, 2, 3, 33, 70, 252, 94, 161, 224, 176, 67, 230,
			5, 35, 22, 242, 24, 12, 180, 116, 20, 41, 177, 214, 81, 74,
			20, 8, 129, 6, 115, 92, 46, 66, 183, 51, 22, 110, 224, 131,
			237, 59, 48, 230, 12, 92, 31, 120, 48, 14, 187, 76, 142, 116,
			92, 223, 14, 143, 165, 94, 60, 15, 71, 174, 24, 64, 16, 202,
			255, 7, 99, 65, 96, 24, 56, 110, 207, 237, 218, 200, 33, 15,
			118, 200, 96, 196, 194, 161, 43, 4, 115, 96, 20, 6, 135, 174,
			195, 28, 16, 3, 91, 128, 24, 160, 117, 158, 23, 28, 185, 126,
			31, 208, 149, 46, 46, 226, 184, 136, 192, 144, 137, 123, 132, 0,
			254, 125, 116, 66, 49, 14, 65, 47, 214, 168, 27, 56, 12, 134,
			99, 46, 32, 100, 194, 118, 125, 201, 213, 238, 4, 135, 248, 42,
			66, 140, 128, 31, 8, 183, 203, 242, 32, 6, 46, 7, 207, 229,
			2, 57, 76, 75, 244, 157, 19, 234, 56, 46, 239, 122, 182, 59,
			100, 161, 245, 38, 37, 92, 127, 26, 139, 88, 137, 81, 24, 56,
			227, 46, 155, 232, 65, 38, 138, 252, 191, 244, 32, 16, 89, 231,
			4, 221, 241, 144, 249, 194, 142, 157, 84, 8, 66, 8, 196, 128,
			133, 48, 180, 5, 11, 93, 219, 227, 19, 168, 165, 131, 196, 128,
			17, 152, 214, 62, 49, 170, 198, 92, 185, 18, 25, 251, 246, 144,
			161, 66, 211, 177, 229, 7, 147, 119, 18, 119, 87, 112, 180, 200,
			87, 172, 130, 144, 195, 208, 62, 134, 14, 195, 72, 113, 64, 4,
			192, 124, 39, 8, 57, 195, 160, 24, 133, 193, 48, 16, 12, 20,
			38, 130, 131, 195, 66, 247, 144, 57, 208, 11, 131, 33, 81, 40,
			240, 160, 39, 142, 48, 76, 162, 8, 2, 62, 98, 93, 140, 32,
			24, 133, 46, 6, 86, 136, 177, 227, 171, 40, 226, 92, 234, 78,
			160, 245, 176, 218, 132, 102, 253, 65, 235, 73, 177, 81, 129, 106,
			19, 246, 27, 245, 199, 213, 114, 165, 12, 219, 79, 161, 245, 176,
			2, 165, 250, 254, 211, 70, 117, 231, 97, 11, 30, 214, 119, 203,
			149, 70, 19, 138, 181, 50, 148, 234, 181, 86, 163, 186, 221, 110,
			213, 27, 77, 2, 217, 98, 19, 170, 205, 172, 124, 83, 172, 61,
			133, 202, 151, 251, 141, 74, 179, 9, 245, 6, 84, 247, 246, 119,
			171, 149, 50, 60, 41, 54, 26, 197, 90, 171, 90, 105, 230, 161,
			90, 43, 237, 182, 203, 213, 218, 78, 30, 182, 219, 45, 168, 213,
			91, 4, 118, 171, 123, 213, 86, 165, 12, 173, 122, 94, 138, 61,
			189, 14, 234, 15, 96, 175, 210, 40, 61, 44, 214, 90, 197, 237,
			234, 110, 181, 245, 84, 10, 124, 80, 109, 213, 80, 216, 131, 122,
			131, 64, 17, 246, 139, 141, 86, 181, 212, 222, 45, 54, 96, 191,
			221, 216, 175, 55, 43, 128, 150, 149, 171, 205, 210, 110, 177, 186,
			87, 41, 91, 80, 173, 65, 173, 14, 149, 199, 149, 90, 11, 154,
			15, 139, 187, 187, 179, 134, 18, 168, 63, 169, 85, 26, 168, 253,
			180, 153, 176, 93, 129, 221, 106, 113, 123, 183, 130, 162, 164, 157,
			229, 106, 163, 82, 106, 161, 65, 147, 167, 82, 181, 92, 169, 181,
			138, 187, 121, 2, 205, 253, 74, 169, 90, 220, 205, 67, 229, 203,
			202, 222, 254, 110, 177, 241, 52, 31, 49, 109, 86, 254, 160, 93,
			169, 181, 170, 197, 93, 40, 23, 247, 138, 59, 149, 38, 228, 222,
			134, 202, 126, 163, 94, 106, 55, 42, 123, 168, 117, 253, 1, 52,
			219, 219, 205, 86, 181, 213, 110, 85, 96, 167, 94, 47, 75, 176,
			155, 149, 198, 227, 106, 169, 210, 188, 15, 187, 245, 166, 4, 172,
			221, 172, 228, 9, 148, 139, 173, 162, 20, 189, 223, 168, 63, 168,
			182, 154, 247, 241, 121, 187, 221, 172, 74, 224, 170, 181, 86, 165,
			209, 104, 239, 183, 170, 245, 218, 42, 60, 172, 63, 169, 60, 174,
			52, 160, 84, 108, 55, 43, 101, 137, 112, 189, 134, 214, 98, 172,
			84, 234, 141, 167, 200, 22, 113, 144, 30, 200, 195, 147, 135, 149,
			214, 195, 74, 3, 65, 149, 104, 21, 17, 134, 102, 171, 81, 45,
			181, 166, 167, 213, 27, 208, 170, 55, 90, 100, 202, 78, 168, 85,
			118, 118, 171, 59, 149, 90, 169, 130, 175, 235, 200, 230, 73, 181,
			89, 89, 133, 98, 163, 218, 196, 9, 85, 41, 24, 158, 20, 159,
			66, 189, 45, 173, 70, 71, 181, 155, 21, 162, 158, 167, 66, 55,
			47, 253, 9, 213, 7, 80, 44, 63, 174, 162, 230, 209, 236, 253,
			122, 179, 89, 141, 194, 69, 194, 86, 122, 24, 97, 110, 17, 146,
			38, 154, 78, 13, 72, 95, 198, 167, 52, 53, 178, 169, 251, 100,
			145, 232, 233, 21, 245, 168, 6, 111, 165, 110, 200, 193, 27, 234,
			81, 13, 126, 144, 218, 150, 131, 75, 234, 81, 13, 174, 164, 242,
			114, 80, 83, 143, 106, 240, 71, 169, 130, 28, 140, 30, 213, 224,
			135, 169, 172, 28, 36, 234, 81, 13, 230, 82, 55, 229, 224, 7,
			234, 241, 159, 174, 18, 221, 76, 209, 185, 111, 176, 242, 101, 126,
			121, 21, 138, 144, 148, 92, 153, 31, 25, 103, 190, 224, 96, 195,
			40, 112, 125, 33, 179, 154, 59, 196, 42, 227, 176, 17, 243, 29,
			230, 203, 172, 104, 251, 199, 106, 252, 155, 192, 103, 4, 179, 73,
			215, 246, 152, 239, 216, 97, 126, 194, 133, 57, 96, 115, 136, 250,
			0, 153, 61, 123, 161, 221, 157, 212, 136, 248, 5, 150, 0, 108,
			10, 36, 141, 53, 50, 240, 84, 137, 115, 125, 104, 183, 74, 80,
			25, 5, 221, 129, 20, 103, 65, 85, 128, 203, 129, 249, 88, 89,
			176, 254, 97, 22, 150, 249, 115, 63, 12, 60, 54, 18, 110, 23,
			118, 66, 214, 15, 66, 215, 246, 161, 20, 233, 4, 71, 3, 183,
			59, 0, 246, 181, 96, 40, 16, 51, 230, 100, 82, 172, 56, 129,
			142, 221, 125, 117, 100, 135, 56, 35, 128, 99, 102, 135, 16, 248,
			167, 68, 218, 156, 143, 135, 40, 213, 246, 60, 24, 186, 254, 88,
			48, 89, 19, 225, 206, 58, 73, 76, 242, 2, 191, 159, 7, 215,
			98, 22, 120, 204, 30, 77, 76, 13, 25, 100, 249, 144, 217, 33,
			115, 178, 192, 3, 85, 106, 253, 96, 122, 22, 1, 97, 119, 60,
			134, 50, 125, 198, 80, 100, 47, 8, 85, 211, 49, 194, 42, 42,
			11, 4, 52, 100, 251, 225, 242, 40, 89, 175, 175, 175, 111, 172,
			201, 255, 90, 235, 235, 247, 228, 127, 207, 208, 138, 187, 119, 239,
			222, 93, 219, 216, 92, 219, 218, 104, 109, 110, 221, 187, 125, 247,
			222, 237, 187, 214, 221, 248, 239, 153, 69, 96, 251, 24, 1, 23,
			161, 219, 21, 18, 202, 72, 165, 16, 217, 231, 225, 136, 1, 243,
			249, 56, 100, 106, 244, 136, 65, 23, 17, 11, 252, 67, 22, 10,
			16, 1, 137, 188, 26, 12, 1, 26, 15, 74, 176, 181, 181, 117,
			23, 155, 36, 6, 200, 210, 239, 115, 139, 64, 147, 49, 120, 30,
			119, 59, 71, 71, 71, 150, 203, 68, 207, 10, 194, 126, 33, 236,
			117, 241, 31, 46, 178, 196, 215, 226, 171, 220, 111, 51, 107, 21,
			11, 204, 45, 168, 124, 109, 15, 71, 30, 227, 132, 196, 143, 176,
			113, 15, 74, 193, 112, 52, 22, 108, 42, 164, 165, 110, 251, 245,
			102, 245, 75, 56, 192, 8, 202, 173, 30, 88, 81, 227, 50, 153,
			148, 52, 144, 247, 213, 155, 73, 235, 203, 153, 120, 17, 57, 47,
			39, 151, 215, 218, 187, 187, 171, 171, 175, 157, 39, 99, 56, 183,
			190, 122, 127, 74, 167, 205, 183, 233, 212, 103, 2, 185, 4, 61,
			199, 62, 158, 210, 141, 139, 112, 220, 21, 82, 192, 161, 237, 129,
			56, 140, 36, 206, 76, 255, 145, 56, 204, 131, 84, 232, 254, 239,
			106, 210, 161, 37, 14, 145, 250, 77, 22, 169, 73, 99, 206, 186,
			240, 17, 108, 172, 175, 207, 90, 184, 245, 70, 11, 159, 184, 254,
			214, 38, 28, 236, 48, 209, 60, 230, 130, 13, 241, 117, 145, 63,
			112, 61, 214, 154, 117, 196, 131, 234, 110, 165, 85, 221, 171, 64,
			79, 68, 106, 188, 105, 205, 143, 122, 34, 214, 180, 93, 173, 181,
			238, 124, 12, 194, 237, 190, 226, 240, 25, 228, 114, 57, 53, 178,
			218, 19, 150, 115, 244, 208, 237, 15, 202, 182, 144, 171, 86, 225,
			211, 79, 97, 107, 115, 21, 254, 16, 228, 187, 221, 224, 40, 126,
			21, 227, 86, 40, 64, 17, 245, 117, 130, 35, 46, 89, 226, 206,
			218, 88, 95, 159, 202, 75, 220, 74, 38, 48, 153, 143, 54, 238,
			156, 222, 114, 9, 55, 92, 190, 113, 231, 227, 143, 63, 254, 100,
			235, 206, 250, 122, 178, 255, 59, 172, 23, 132, 12, 218, 190, 251,
			117, 204, 229, 238, 39, 235, 39, 185, 88, 191, 155, 51, 115, 202,
			126, 200, 229, 20, 40, 5, 233, 44, 252, 91, 133, 181, 105, 117,
			222, 18, 193, 200, 7, 225, 138, 249, 172, 76, 241, 145, 1, 176,
			58, 19, 0, 31, 191, 49, 0, 30, 217, 135, 54, 28, 40, 71,
			90, 221, 113, 24, 50, 95, 224, 148, 61, 215, 243, 92, 62, 21,
			0, 152, 46, 97, 40, 71, 225, 51, 120, 243, 130, 223, 16, 230,
			240, 217, 100, 212, 242, 217, 209, 246, 216, 245, 28, 22, 230, 86,
			209, 176, 102, 132, 80, 36, 66, 1, 179, 170, 120, 225, 31, 206,
			169, 41, 219, 93, 95, 160, 229, 209, 76, 101, 122, 100, 182, 68,
			96, 213, 234, 32, 103, 169, 203, 4, 131, 219, 111, 196, 32, 178,
			34, 46, 162, 176, 127, 44, 6, 170, 73, 158, 129, 127, 90, 253,
			220, 234, 73, 223, 236, 48, 81, 154, 160, 145, 91, 149, 25, 240,
			81, 179, 94, 131, 61, 123, 52, 114, 253, 62, 33, 80, 245, 213,
			136, 58, 145, 230, 101, 145, 155, 194, 233, 120, 196, 102, 171, 24,
			216, 81, 142, 142, 14, 46, 4, 158, 199, 25, 252, 183, 76, 196,
			145, 40, 11, 90, 88, 27, 92, 158, 87, 108, 212, 40, 10, 203,
			126, 139, 69, 244, 187, 181, 111, 135, 129, 47, 6, 223, 173, 125,
			235, 216, 199, 223, 181, 190, 29, 4, 227, 240, 187, 123, 223, 14,
			93, 255, 187, 123, 223, 114, 214, 253, 238, 185, 245, 45, 54, 6,
			24, 200, 223, 125, 245, 44, 75, 224, 104, 192, 66, 6, 106, 53,
			50, 178, 189, 35, 251, 152, 3, 251, 26, 27, 11, 158, 212, 253,
			94, 48, 14, 193, 113, 251, 174, 224, 88, 225, 61, 6, 145, 164,
			60, 72, 81, 121, 2, 74, 88, 30, 164, 180, 188, 172, 86, 82,
			164, 172, 196, 223, 176, 48, 88, 27, 217, 142, 163, 142, 70, 226,
			40, 136, 185, 49, 187, 59, 64, 187, 88, 210, 177, 216, 94, 82,
			221, 243, 81, 59, 129, 165, 176, 31, 192, 120, 36, 11, 109, 188,
			52, 39, 171, 190, 26, 220, 120, 125, 95, 179, 154, 39, 82, 126,
			48, 82, 156, 149, 164, 236, 179, 44, 240, 113, 175, 231, 126, 141,
			205, 22, 158, 209, 153, 106, 85, 48, 14, 176, 205, 130, 92, 182,
			221, 42, 101, 87, 239, 207, 140, 18, 4, 40, 100, 63, 27, 187,
			33, 115, 44, 40, 130, 188, 58, 216, 82, 193, 192, 229, 121, 211,
			253, 134, 133, 192, 7, 193, 216, 115, 98, 40, 199, 156, 201, 214,
			42, 103, 243, 68, 154, 3, 157, 99, 130, 106, 172, 162, 3, 124,
			60, 225, 249, 34, 234, 175, 78, 134, 18, 2, 105, 207, 136, 26,
			217, 33, 159, 136, 233, 48, 2, 178, 139, 17, 1, 216, 221, 46,
			27, 9, 232, 4, 98, 32, 101, 226, 90, 117, 32, 142, 109, 224,
			167, 244, 0, 219, 135, 160, 215, 227, 76, 213, 251, 7, 65, 8,
			76, 237, 181, 60, 100, 55, 215, 55, 62, 193, 156, 185, 113, 187,
			181, 190, 113, 111, 107, 253, 222, 198, 109, 107, 125, 227, 89, 54,
			138, 110, 14, 146, 78, 146, 238, 200, 230, 130, 128, 156, 41, 229,
			7, 62, 60, 178, 253, 177, 29, 30, 195, 198, 237, 60, 32, 55,
			43, 218, 64, 246, 161, 221, 236, 134, 238, 72, 228, 177, 245, 155,
			105, 118, 108, 192, 162, 1, 65, 231, 37, 195, 194, 28, 168, 243,
			113, 20, 236, 83, 125, 40, 23, 54, 118, 147, 14, 60, 23, 65,
			181, 89, 111, 202, 61, 150, 91, 157, 236, 169, 228, 194, 199, 26,
			6, 223, 184, 158, 103, 203, 205, 197, 252, 181, 118, 179, 224, 4,
			93, 94, 120, 194, 58, 133, 137, 38, 133, 6, 235, 177, 144, 249,
			93, 86, 216, 241, 130, 142, 237, 189, 168, 75, 21, 120, 1, 245,
			41, 76, 9, 249, 74, 94, 203, 12, 2, 199, 66, 91, 84, 162,
			201, 203, 109, 30, 105, 116, 128, 157, 153, 108, 163, 227, 135, 131,
			216, 30, 180, 180, 195, 98, 99, 25, 54, 161, 175, 179, 240, 249,
			1, 23, 97, 79, 174, 156, 50, 40, 232, 114, 107, 164, 242, 26,
			154, 178, 89, 240, 220, 78, 104, 135, 199, 242, 98, 206, 26, 136,
			161, 119, 75, 62, 197, 107, 87, 73, 114, 239, 161, 242, 98, 36,
			131, 143, 88, 23, 62, 92, 121, 186, 182, 50, 92, 91, 113, 90,
			43, 15, 239, 173, 236, 221, 91, 105, 90, 43, 189, 103, 31, 90,
			176, 235, 190, 98, 71, 46, 103, 121, 76, 88, 136, 143, 244, 17,
			145, 170, 99, 56, 35, 183, 71, 129, 99, 203, 80, 253, 144, 195,
			243, 131, 106, 179, 30, 23, 250, 7, 42, 85, 57, 17, 153, 91,
			61, 248, 42, 167, 238, 224, 162, 44, 247, 50, 112, 148, 35, 240,
			97, 13, 181, 42, 216, 35, 87, 250, 35, 30, 149, 230, 20, 148,
			174, 133, 211, 188, 165, 157, 177, 128, 181, 53, 2, 171, 136, 97,
			208, 145, 247, 94, 118, 100, 163, 96, 120, 82, 26, 201, 173, 17,
			244, 160, 207, 124, 22, 218, 106, 147, 197, 27, 140, 171, 132, 156,
			64, 111, 17, 249, 103, 152, 41, 141, 26, 223, 164, 151, 201, 95,
			107, 196, 52, 83, 122, 138, 26, 223, 235, 239, 100, 254, 92, 131,
			198, 228, 216, 22, 199, 124, 208, 147, 161, 46, 209, 229, 174, 223,
			157, 238, 57, 200, 235, 155, 14, 216, 27, 115, 129, 65, 32, 235,
			214, 27, 14, 20, 228, 117, 39, 138, 103, 224, 250, 93, 111, 204,
			221, 67, 102, 17, 114, 150, 204, 161, 118, 38, 53, 191, 215, 191,
			185, 72, 206, 40, 114, 14, 181, 93, 136, 41, 141, 26, 223, 167,
			207, 199, 148, 65, 141, 239, 233, 69, 242, 95, 202, 46, 141, 154,
			63, 215, 116, 154, 249, 119, 13, 106, 129, 191, 230, 179, 190, 45,
			220, 67, 54, 123, 118, 180, 35, 75, 1, 143, 79, 175, 203, 177,
			22, 212, 162, 133, 113, 222, 134, 67, 219, 27, 51, 174, 66, 111,
			194, 76, 94, 12, 114, 225, 122, 30, 12, 236, 67, 6, 254, 180,
			76, 201, 58, 90, 72, 212, 25, 168, 27, 140, 125, 129, 174, 193,
			147, 98, 124, 60, 62, 9, 94, 116, 244, 202, 71, 255, 200, 12,
			64, 231, 164, 213, 154, 73, 231, 126, 174, 233, 223, 191, 19, 1,
			166, 205, 73, 187, 23, 98, 82, 194, 144, 62, 27, 147, 6, 146,
			23, 150, 59, 243, 42, 231, 146, 255, 180, 200, 154, 235, 247, 66,
			187, 96, 143, 70, 204, 239, 187, 62, 43, 56, 97, 224, 179, 181,
			159, 141, 25, 243, 49, 120, 11, 156, 133, 135, 110, 55, 186, 93,
			167, 75, 242, 245, 11, 249, 58, 243, 182, 219, 254, 236, 127, 107,
			132, 54, 216, 40, 8, 69, 25, 151, 53, 216, 207, 198, 140, 11,
			250, 62, 33, 138, 205, 120, 236, 58, 242, 166, 127, 177, 177, 40,
			71, 218, 99, 215, 161, 79, 200, 121, 47, 176, 157, 23, 81, 42,
			15, 66, 117, 235, 191, 180, 105, 89, 83, 210, 173, 211, 140, 173,
			221, 192, 118, 170, 201, 170, 198, 57, 111, 134, 166, 63, 38, 203,
			138, 129, 195, 184, 76, 139, 110, 224, 95, 49, 164, 248, 11, 242,
			69, 121, 50, 158, 217, 34, 231, 102, 217, 209, 155, 228, 140, 51,
			22, 47, 112, 231, 117, 93, 113, 44, 21, 63, 219, 88, 114, 198,
			162, 20, 13, 101, 255, 69, 39, 23, 103, 244, 226, 163, 192, 231,
			140, 126, 78, 230, 185, 176, 197, 88, 125, 215, 56, 183, 249, 225,
			155, 45, 81, 43, 172, 166, 156, 222, 136, 150, 157, 128, 76, 63,
			9, 89, 137, 156, 103, 95, 143, 220, 80, 158, 235, 95, 160, 27,
			164, 93, 75, 155, 153, 147, 31, 71, 172, 164, 6, 55, 206, 77,
			150, 224, 32, 189, 69, 206, 218, 156, 187, 125, 159, 57, 47, 156,
			177, 224, 87, 76, 48, 114, 139, 141, 51, 241, 96, 121, 44, 56,
			78, 114, 66, 219, 245, 93, 191, 175, 38, 205, 169, 73, 241, 32,
			78, 202, 222, 38, 243, 74, 127, 186, 76, 206, 182, 107, 95, 212,
			234, 79, 106, 47, 42, 141, 70, 189, 113, 33, 69, 231, 137, 94,
			255, 226, 130, 70, 47, 144, 51, 241, 171, 118, 187, 90, 190, 160,
			103, 119, 48, 90, 60, 102, 115, 134, 92, 126, 203, 104, 161, 196,
			148, 122, 232, 82, 15, 249, 156, 125, 23, 189, 48, 197, 72, 97,
			154, 205, 17, 90, 102, 93, 207, 14, 103, 248, 199, 12, 180, 89,
			6, 51, 51, 35, 6, 23, 201, 242, 174, 203, 149, 167, 226, 245,
			217, 255, 208, 8, 157, 30, 141, 92, 254, 25, 153, 151, 74, 42,
			198, 75, 155, 43, 51, 46, 63, 189, 192, 82, 254, 143, 22, 101,
			126, 161, 145, 57, 57, 66, 207, 17, 61, 177, 91, 127, 189, 175,
			245, 31, 236, 235, 31, 178, 21, 178, 203, 228, 188, 212, 119, 2,
			90, 246, 239, 53, 114, 97, 50, 22, 153, 124, 123, 10, 201, 165,
			205, 155, 167, 13, 158, 154, 108, 149, 199, 66, 129, 157, 249, 146,
			24, 229, 177, 56, 101, 231, 10, 57, 55, 9, 71, 228, 20, 133,
			125, 18, 164, 10, 158, 12, 73, 199, 177, 39, 13, 72, 55, 18,
			122, 243, 31, 18, 16, 247, 201, 210, 212, 46, 163, 55, 222, 146,
			73, 50, 240, 182, 13, 170, 56, 38, 49, 118, 138, 227, 201, 48,
			62, 197, 241, 84, 120, 110, 50, 178, 92, 245, 15, 153, 47, 130,
			240, 120, 95, 125, 219, 9, 81, 204, 84, 36, 158, 16, 115, 58,
			154, 79, 136, 121, 77, 16, 111, 254, 173, 70, 22, 170, 62, 182,
			73, 130, 238, 17, 50, 137, 68, 122, 253, 141, 33, 170, 120, 223,
			120, 75, 8, 211, 29, 146, 142, 253, 76, 175, 189, 193, 253, 138,
			213, 251, 191, 49, 56, 182, 231, 158, 25, 246, 200, 125, 244, 207,
			55, 201, 60, 53, 205, 84, 160, 145, 95, 107, 242, 243, 170, 153,
			162, 155, 191, 210, 102, 190, 148, 110, 220, 149, 39, 159, 221, 118,
			169, 10, 197, 177, 24, 4, 33, 183, 222, 240, 185, 180, 205, 101,
			195, 20, 125, 148, 154, 124, 92, 116, 57, 244, 131, 67, 22, 250,
			120, 42, 244, 157, 232, 91, 89, 113, 100, 119, 145, 177, 219, 101,
			62, 118, 140, 143, 89, 200, 221, 192, 135, 77, 107, 61, 46, 227,
			170, 225, 237, 5, 99, 223, 137, 63, 221, 237, 86, 75, 149, 90,
			179, 2, 61, 215, 195, 58, 189, 72, 116, 35, 69, 141, 249, 133,
			92, 116, 165, 159, 78, 95, 140, 46, 213, 73, 42, 19, 95, 212,
			227, 35, 33, 250, 124, 138, 154, 103, 82, 151, 52, 236, 212, 230,
			177, 191, 57, 147, 62, 75, 126, 169, 17, 115, 94, 118, 106, 84,
			47, 103, 254, 74, 118, 106, 113, 52, 162, 230, 93, 219, 243, 212,
			113, 71, 165, 14, 108, 27, 66, 57, 5, 60, 247, 144, 249, 140,
			171, 107, 243, 62, 19, 80, 110, 183, 8, 168, 253, 51, 196, 86,
			15, 143, 44, 77, 166, 218, 222, 70, 165, 88, 222, 171, 200, 123,
			98, 135, 9, 219, 245, 56, 30, 114, 240, 141, 252, 20, 104, 227,
			129, 37, 254, 166, 43, 37, 201, 174, 135, 68, 31, 50, 45, 130,
			141, 216, 188, 106, 203, 232, 252, 114, 76, 233, 212, 160, 244, 131,
			152, 50, 168, 65, 11, 219, 100, 87, 90, 164, 81, 227, 93, 189,
			156, 249, 28, 166, 54, 195, 155, 13, 146, 83, 32, 56, 242, 89,
			200, 7, 238, 8, 253, 88, 110, 183, 120, 34, 87, 67, 118, 137,
			92, 68, 250, 221, 68, 174, 102, 80, 227, 221, 194, 182, 132, 88,
			163, 230, 149, 212, 53, 5, 49, 174, 185, 146, 126, 143, 116, 136,
			57, 175, 33, 194, 87, 245, 114, 166, 13, 83, 187, 6, 4, 243,
			60, 117, 130, 142, 26, 33, 176, 59, 193, 88, 200, 139, 123, 25,
			74, 76, 170, 1, 246, 161, 237, 122, 209, 97, 53, 134, 24, 21,
			87, 38, 68, 90, 106, 18, 157, 171, 145, 150, 154, 68, 231, 106,
			164, 165, 38, 209, 185, 90, 216, 38, 127, 169, 17, 125, 94, 167,
			38, 164, 110, 105, 153, 95, 104, 16, 109, 214, 68, 129, 232, 187,
			47, 135, 198, 126, 137, 79, 238, 245, 177, 23, 61, 100, 224, 170,
			217, 110, 224, 23, 28, 214, 25, 247, 251, 174, 223, 183, 8, 110,
			17, 206, 212, 138, 168, 67, 77, 190, 84, 64, 55, 24, 142, 108,
			225, 118, 92, 207, 21, 199, 16, 132, 120, 216, 139, 136, 254, 216,
			14, 109, 95, 48, 105, 2, 66, 134, 94, 131, 244, 121, 178, 68,
			204, 121, 29, 33, 187, 169, 23, 165, 254, 186, 180, 237, 230, 252,
			133, 152, 210, 169, 113, 115, 57, 27, 83, 6, 53, 110, 174, 125,
			30, 45, 211, 168, 145, 213, 239, 71, 175, 208, 9, 217, 249, 115,
			49, 165, 83, 35, 123, 254, 122, 76, 25, 212, 200, 174, 222, 69,
			199, 153, 41, 106, 174, 164, 54, 181, 228, 20, 179, 146, 206, 144,
			63, 75, 78, 49, 57, 253, 74, 230, 123, 152, 52, 10, 24, 72,
			232, 28, 108, 45, 32, 174, 24, 234, 60, 26, 133, 175, 5, 80,
			99, 71, 113, 140, 169, 43, 7, 2, 30, 67, 116, 100, 134, 96,
			195, 145, 56, 190, 15, 54, 248, 236, 72, 241, 57, 194, 6, 191,
			195, 222, 192, 111, 250, 212, 146, 211, 87, 174, 77, 157, 90, 114,
			122, 122, 234, 212, 146, 91, 188, 56, 117, 106, 201, 93, 186, 76,
			238, 71, 135, 22, 227, 35, 125, 37, 99, 193, 137, 246, 87, 222,
			243, 200, 111, 239, 232, 108, 124, 9, 29, 219, 179, 253, 174, 116,
			109, 220, 220, 155, 212, 252, 72, 207, 93, 137, 56, 107, 243, 200,
			236, 66, 76, 33, 235, 101, 136, 41, 131, 26, 31, 221, 250, 128,
			60, 150, 82, 117, 106, 228, 245, 27, 153, 42, 156, 106, 8, 228,
			173, 25, 12, 198, 67, 219, 135, 94, 232, 50, 223, 241, 142, 97,
			250, 125, 180, 1, 226, 235, 201, 89, 24, 116, 147, 154, 121, 253,
			163, 149, 72, 168, 62, 135, 114, 98, 24, 208, 214, 252, 98, 38,
			166, 12, 106, 228, 223, 71, 167, 155, 102, 202, 72, 81, 115, 77,
			223, 48, 212, 59, 3, 1, 91, 35, 87, 8, 39, 243, 72, 161,
			175, 215, 205, 107, 25, 7, 166, 123, 113, 165, 41, 119, 229, 61,
			170, 4, 40, 65, 79, 37, 173, 201, 101, 216, 32, 56, 130, 161,
			237, 31, 19, 16, 129, 176, 61, 181, 123, 39, 57, 13, 83, 58,
			31, 143, 48, 125, 90, 132, 156, 39, 11, 74, 168, 73, 205, 117,
			115, 45, 67, 206, 197, 3, 115, 168, 6, 153, 208, 26, 53, 214,
			151, 46, 79, 104, 131, 26, 235, 153, 171, 50, 112, 53, 106, 110,
			165, 190, 80, 129, 139, 142, 216, 74, 95, 37, 54, 49, 77, 153,
			113, 238, 232, 239, 100, 90, 160, 186, 252, 40, 109, 71, 233, 70,
			13, 197, 16, 219, 158, 103, 65, 244, 21, 209, 29, 226, 52, 219,
			151, 55, 72, 221, 1, 235, 190, 34, 201, 89, 31, 88, 24, 98,
			5, 84, 94, 208, 164, 230, 119, 244, 173, 247, 37, 154, 154, 158,
			154, 71, 145, 233, 152, 210, 168, 113, 103, 241, 124, 76, 25, 212,
			184, 67, 47, 74, 47, 104, 184, 221, 62, 209, 63, 87, 94, 208,
			228, 134, 251, 100, 225, 44, 249, 99, 157, 204, 35, 137, 170, 127,
			106, 94, 202, 252, 143, 6, 51, 253, 125, 124, 117, 231, 7, 34,
			249, 193, 136, 31, 132, 67, 219, 243, 142, 19, 253, 37, 218, 172,
			103, 143, 61, 65, 212, 185, 24, 220, 222, 180, 209, 46, 7, 249,
			67, 16, 191, 143, 217, 104, 236, 191, 242, 131, 35, 223, 130, 217,
			27, 60, 181, 132, 36, 105, 113, 204, 25, 143, 54, 43, 243, 199,
			195, 136, 113, 82, 178, 186, 158, 43, 99, 52, 96, 92, 106, 135,
			60, 73, 148, 204, 143, 89, 116, 215, 29, 77, 146, 251, 60, 190,
			14, 138, 52, 85, 252, 44, 229, 114, 45, 218, 201, 159, 154, 203,
			19, 90, 167, 198, 167, 239, 188, 75, 206, 70, 8, 105, 212, 248,
			204, 92, 74, 94, 107, 146, 158, 159, 208, 58, 53, 62, 91, 36,
			201, 116, 157, 26, 63, 49, 223, 77, 94, 227, 242, 159, 152, 23,
			38, 52, 190, 191, 248, 14, 249, 27, 77, 70, 142, 70, 141, 146,
			126, 37, 243, 23, 218, 15, 77, 121, 213, 222, 244, 138, 35, 155,
			35, 128, 34, 110, 94, 66, 213, 158, 69, 191, 94, 234, 185, 204,
			115, 20, 24, 209, 79, 136, 6, 42, 57, 50, 224, 246, 144, 69,
			8, 7, 33, 65, 87, 7, 234, 7, 96, 73, 224, 97, 62, 42,
			37, 1, 164, 107, 115, 168, 113, 28, 120, 8, 70, 41, 202, 130,
			154, 204, 71, 165, 75, 151, 201, 3, 105, 154, 78, 141, 138, 190,
			158, 185, 11, 39, 78, 57, 104, 158, 188, 82, 158, 78, 57, 147,
			94, 70, 77, 103, 147, 200, 199, 252, 83, 209, 75, 87, 34, 33,
			250, 60, 242, 189, 26, 83, 26, 53, 42, 215, 126, 28, 83, 6,
			53, 42, 86, 129, 252, 190, 84, 192, 160, 198, 142, 254, 65, 102,
			11, 102, 78, 195, 50, 9, 79, 202, 253, 27, 42, 128, 226, 103,
			152, 200, 34, 161, 230, 168, 177, 179, 180, 28, 83, 26, 53, 118,
			232, 141, 152, 66, 97, 217, 91, 36, 148, 146, 77, 106, 60, 210,
			63, 200, 32, 187, 169, 35, 246, 172, 228, 19, 61, 88, 180, 223,
			228, 2, 11, 160, 133, 126, 115, 57, 137, 175, 232, 109, 224, 227,
			14, 58, 56, 232, 205, 154, 147, 232, 106, 74, 161, 9, 53, 71,
			141, 71, 137, 174, 166, 70, 141, 71, 137, 174, 166, 65, 141, 71,
			217, 91, 50, 167, 233, 212, 220, 75, 53, 85, 78, 67, 44, 247,
			210, 25, 242, 41, 49, 77, 217, 18, 212, 245, 43, 153, 194, 15,
			11, 76, 229, 52, 93, 166, 171, 186, 190, 167, 106, 167, 46, 211,
			108, 61, 138, 26, 213, 96, 212, 163, 168, 81, 45, 69, 253, 210,
			101, 242, 92, 138, 213, 168, 209, 208, 175, 102, 106, 32, 17, 155,
			116, 140, 73, 210, 193, 61, 111, 251, 42, 61, 98, 238, 176, 17,
			206, 228, 197, 68, 41, 114, 202, 159, 58, 198, 178, 209, 208, 19,
			106, 142, 26, 141, 8, 35, 213, 190, 52, 232, 165, 152, 50, 168,
			209, 120, 79, 246, 245, 8, 87, 43, 117, 93, 66, 132, 78, 111,
			165, 85, 57, 48, 169, 249, 56, 245, 76, 65, 135, 0, 63, 78,
			103, 200, 159, 224, 174, 54, 17, 187, 167, 250, 213, 140, 80, 70,
			200, 111, 86, 179, 93, 166, 8, 112, 171, 13, 109, 135, 205, 52,
			156, 113, 151, 9, 114, 22, 137, 247, 162, 186, 71, 156, 252, 16,
			49, 110, 95, 100, 176, 48, 71, 158, 10, 28, 230, 49, 181, 109,
			209, 0, 19, 29, 96, 60, 213, 19, 106, 142, 26, 79, 35, 83,
			77, 137, 255, 211, 200, 84, 83, 226, 255, 52, 50, 117, 142, 26,
			207, 35, 83, 231, 52, 106, 60, 79, 95, 149, 195, 243, 212, 248,
			42, 117, 77, 14, 207, 107, 212, 248, 42, 253, 158, 68, 96, 129,
			154, 63, 77, 49, 133, 192, 130, 70, 141, 159, 166, 51, 178, 232,
			44, 96, 233, 127, 161, 119, 85, 206, 88, 144, 165, 255, 5, 57,
			47, 83, 228, 130, 42, 253, 7, 38, 149, 69, 121, 33, 42, 202,
			7, 230, 11, 149, 129, 23, 162, 162, 124, 16, 21, 229, 133, 168,
			40, 31, 44, 157, 157, 208, 6, 53, 14, 46, 44, 39, 252, 52,
			106, 216, 230, 102, 194, 15, 51, 150, 109, 30, 208, 100, 62, 246,
			80, 182, 249, 254, 132, 198, 5, 215, 215, 38, 180, 65, 13, 123,
			125, 35, 225, 167, 83, 163, 99, 222, 76, 248, 97, 2, 234, 152,
			246, 102, 50, 31, 91, 160, 206, 148, 126, 168, 64, 103, 233, 218,
			132, 54, 168, 209, 185, 1, 216, 36, 155, 11, 104, 173, 163, 171,
			141, 176, 32, 221, 226, 68, 110, 89, 144, 85, 220, 89, 186, 16,
			83, 26, 53, 156, 229, 203, 49, 101, 80, 195, 201, 40, 252, 211,
			212, 232, 203, 147, 165, 97, 166, 53, 106, 244, 211, 151, 37, 254,
			139, 212, 28, 224, 153, 26, 199, 23, 53, 106, 12, 210, 87, 36,
			254, 139, 136, 191, 171, 15, 21, 254, 139, 18, 127, 151, 156, 149,
			246, 45, 42, 252, 95, 70, 248, 47, 70, 248, 191, 52, 221, 243,
			82, 255, 197, 8, 255, 151, 145, 125, 139, 17, 254, 47, 35, 252,
			23, 35, 252, 95, 70, 248, 47, 42, 252, 95, 153, 215, 19, 126,
			136, 255, 43, 243, 37, 77, 230, 227, 70, 123, 53, 197, 15, 241,
			127, 181, 244, 222, 132, 54, 168, 241, 234, 218, 251, 9, 63, 157,
			26, 158, 121, 41, 225, 135, 248, 123, 230, 171, 235, 201, 124, 196,
			223, 51, 211, 19, 90, 163, 134, 183, 184, 60, 161, 13, 106, 120,
			239, 188, 43, 241, 95, 68, 107, 125, 93, 85, 143, 69, 137, 191,
			31, 225, 191, 40, 241, 247, 151, 206, 197, 148, 70, 13, 255, 252,
			197, 152, 50, 168, 225, 95, 186, 28, 223, 177, 255, 95, 0, 0,
			0, 255, 255, 213, 203, 63, 244, 22, 47, 0, 0},
	)
}

// FileDescriptorSet returns a descriptor set for this proto package, which
// includes all defined services, and all transitive dependencies.
//
// Will not return nil.
//
// Do NOT modify the returned descriptor.
func FileDescriptorSet() *descriptor.FileDescriptorSet {
	// We just need ONE of the service names to look up the FileDescriptorSet.
	ret, err := discovery.GetDescriptorSet("drone_queen.Drone")
	if err != nil {
		panic(err)
	}
	return ret
}
