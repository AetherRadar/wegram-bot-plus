方法一：GitHub 一键部署（推荐 ⭐）
这是最简单的部署方式，无需本地开发环境，直接通过 GitHub 仓库部署。

Fork 或克隆本仓库到您的 GitHub 账户
登录 Cloudflare Dashboard
登录 Cloudflare 仪表板
导航到 Workers & Pages 部分
点击 Create Application
选择 Connect to Git
授权 Cloudflare 访问您的 GitHub，并选择您 fork 的仓库
配置部署设置：
Project name：设置您的项目名称（例如 open-wegram-bot）
Production branch：选择主分支（通常是 master）
其他设置保持默认
配置环境变量：
点击 Environment Variables
添加 PREFIX（例如：public）
添加 SECRET_TOKEN（必须包含大小写字母和数字，长度至少16位），并标记为加密
点击 Save and Deploy 按钮完成部署
这种方式的优点是：当您更新 GitHub 仓库时，Cloudflare 会自动重新部署您的 Worker。
