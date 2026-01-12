from PIL import Image, ImageDraw, ImageFont
import os

def create_canvas():
    """Returns base RGBA image with dark background."""
    width, height = 1200, 630
    image = Image.new('RGBA', (width, height), (30, 30, 30, 255))
    return image

def draw_window(img):
    """Draws rounded rectangle window with title-bar and buttons."""
    draw = ImageDraw.Draw(img)
    
    # Window dimensions
    window_x, window_y = 50, 50
    window_width, window_height = 1100, 530
    corner_radius = 12
    
    # Draw main window background
    window_bg = (40, 40, 40, 255)
    draw.rounded_rectangle(
        [(window_x, window_y), (window_x + window_width, window_y + window_height)],
        radius=corner_radius,
        fill=window_bg
    )
    
    # Draw title bar
    title_bar_height = 40
    title_bar_bg = (50, 50, 50, 255)
    draw.rounded_rectangle(
        [(window_x, window_y), (window_x + window_width, window_y + title_bar_height)],
        radius=corner_radius,
        fill=title_bar_bg
    )
    
    # Draw title bar bottom rectangle to complete the rounded top
    draw.rectangle(
        [(window_x, window_y + corner_radius), (window_x + window_width, window_y + title_bar_height)],
        fill=title_bar_bg
    )
    
    # Draw window control buttons
    button_y = window_y + 12
    button_size = 16
    button_spacing = 8
    
    # Red close button
    close_x = window_x + 16
    draw.ellipse(
        [(close_x, button_y), (close_x + button_size, button_y + button_size)],
        fill=(255, 95, 86, 255)
    )
    
    # Yellow minimize button
    minimize_x = close_x + button_size + button_spacing
    draw.ellipse(
        [(minimize_x, button_y), (minimize_x + button_size, button_y + button_size)],
        fill=(255, 189, 46, 255)
    )
    
    # Green maximize button
    maximize_x = minimize_x + button_size + button_spacing
    draw.ellipse(
        [(maximize_x, button_y), (maximize_x + button_size, button_y + button_size)],
        fill=(39, 201, 63, 255)
    )
    
    return draw

def render_prompt(draw):
    """Writes $ gh select prompt line with green prompt glyph."""
    try:
        # Try to load a monospace font
        font = ImageFont.truetype("Monaco.ttf", 18)
    except:
        try:
            font = ImageFont.truetype("Consolas.ttf", 18)
        except:
            try:
                font = ImageFont.truetype("DejaVuSansMono.ttf", 18)
            except:
                font = ImageFont.load_default()
    
    # Prompt position
    prompt_x = 80
    prompt_y = 120
    
    # Draw green prompt symbol
    prompt_color = (57, 255, 20, 255)  # Bright green
    draw.text((prompt_x, prompt_y), "$ ", font=font, fill=prompt_color)
    
    # Draw command text
    command_color = (220, 220, 220, 255)  # Light gray
    command_text = "gh select"
    
    # Calculate position for command text
    prompt_width = draw.textlength("$ ", font=font)
    draw.text((prompt_x + prompt_width, prompt_y), command_text, font=font, fill=command_color)

def render_fzf(draw, repos, selected_index):
    """Renders fuzzy-search interface with syntax coloring and selection highlighting."""
    try:
        # Try to load fonts
        font = ImageFont.truetype("Monaco.ttf", 16)
        bold_font = ImageFont.truetype("Monaco-Bold.ttf", 16)
    except:
        try:
            font = ImageFont.truetype("Consolas.ttf", 16)
            bold_font = ImageFont.truetype("Consolas-Bold.ttf", 16)
        except:
            try:
                font = ImageFont.truetype("DejaVuSansMono.ttf", 16)
                bold_font = ImageFont.truetype("DejaVuSansMono-Bold.ttf", 16)
            except:
                font = ImageFont.load_default()
                bold_font = font
    
    # FZF header
    header_x = 80
    header_y = 170
    header_color = (0, 255, 255, 255)  # Cyan
    draw.text((header_x, header_y), "QUERY ›", font=font, fill=header_color)
    
    # Repository list
    list_start_y = 220
    line_height = 28
    
    for i, repo in enumerate(repos):
        y_pos = list_start_y + (i * line_height)
        
        # Highlight selected item
        if i == selected_index:
            # Purple background for selected item
            highlight_color = (139, 69, 199, 255)  # Purple
            padding = 8
            draw.rectangle(
                [(header_x - padding, y_pos - 4), (1120, y_pos + line_height - 4)],
                fill=highlight_color
            )
            text_color = (255, 255, 255, 255)  # White text on purple
        else:
            text_color = (220, 220, 220, 255)  # Light gray
        
        # Parse org/repo format
        if '/' in repo:
            org, repo_name = repo.split('/', 1)
            
            # Draw org name
            draw.text((header_x, y_pos), org, font=font, fill=text_color)
            
            # Calculate position for separator
            org_width = draw.textlength(org, font=font)
            separator_x = header_x + org_width
            
            # Draw separator with gray color (or white if selected)
            separator_color = (120, 120, 120, 255) if i != selected_index else (255, 255, 255, 255)
            draw.text((separator_x, y_pos), "/", font=font, fill=separator_color)
            
            # Draw repo name in bold
            separator_width = draw.textlength("/", font=font)
            repo_x = separator_x + separator_width
            draw.text((repo_x, y_pos), repo_name, font=bold_font, fill=text_color)
        else:
            # Draw repo name without organization
            draw.text((header_x, y_pos), repo, font=bold_font, fill=text_color)

def add_branding(img):
    """Adds small GitHub CLI logo bottom-right."""
    draw = ImageDraw.Draw(img)
    
    # Simple GitHub CLI branding text
    try:
        font = ImageFont.truetype("Monaco.ttf", 12)
    except:
        try:
            font = ImageFont.truetype("Consolas.ttf", 12)
        except:
            font = ImageFont.load_default()
    
    # Position in bottom-right corner
    brand_text = "GitHub CLI"
    brand_color = (100, 100, 100, 255)  # Dark gray
    
    # Calculate position
    text_width = draw.textlength(brand_text, font=font)
    brand_x = img.width - text_width - 20
    brand_y = img.height - 30
    
    draw.text((brand_x, brand_y), brand_text, font=font, fill=brand_color)

def main():
    """Orchestrates the image generation and saves the result."""
    # Sample repository data
    repos = [
        "octocat/Hello-World",
        "microsoft/vscode",
        "facebook/react",
        "google/tensorflow",
        "torvalds/linux",
        "nodejs/node",
        "python/cpython",
        "rust-lang/rust"
    ]
    
    selected_index = 2  # facebook/react selected
    
    # Create the image
    img = create_canvas()
    draw = draw_window(img)
    
    # Render components
    render_prompt(draw)
    render_fzf(draw, repos, selected_index)
    add_branding(img)
    
    # Save the image
    img.save("gh-repo-select-og.png", "PNG")
    print("Generated gh-repo-select-og.png")

if __name__ == "__main__":
    main()
