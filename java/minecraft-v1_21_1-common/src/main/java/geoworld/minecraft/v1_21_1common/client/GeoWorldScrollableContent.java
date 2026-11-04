package geoworld.minecraft.v1_21_1common.client;

import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.client.gui.components.AbstractContainerWidget;
import net.minecraft.client.gui.components.AbstractWidget;
import net.minecraft.client.gui.layouts.LinearLayout;
import net.minecraft.client.gui.narration.NarratedElementType;
import net.minecraft.client.gui.narration.NarrationElementOutput;
import net.minecraft.network.chat.Component;
import net.minecraft.util.Mth;

import java.util.ArrayList;
import java.util.List;

/**
 * 1.21.1 does not provide the ScrollableLayout used by newer Minecraft
 * versions. This container keeps the shared GeoWorld form as one normal
 * LinearLayout, but gives it a clipped viewport with real scrolling.
 */
final class GeoWorldScrollableContent extends AbstractContainerWidget {
    private static final int SCROLLBAR_WIDTH = 6;
    private static final int SCROLL_STEP = 24;
    private static final int CONTENT_GAP = 4;

    private final LinearLayout body;
    private final List<AbstractWidget> children = new ArrayList<>();
    private double scrollAmount;
    private boolean draggingScrollbar;

    GeoWorldScrollableContent(
        LinearLayout body,
        int viewportHeight,
        int contentWidth
    ) {
        super(
            0,
            0,
            contentWidth + SCROLLBAR_WIDTH + CONTENT_GAP,
            Math.max(40, viewportHeight),
            Component.translatable("geoworld.create.parameters")
        );
        this.body = body;
        this.body.arrangeElements();
        this.body.visitWidgets(this.children::add);
        this.repositionBody();
    }

    @Override
    public List<? extends net.minecraft.client.gui.components.events.GuiEventListener>
        children() {
        return this.children;
    }

    @Override
    public void setX(int x) {
        super.setX(x);
        this.repositionBody();
    }

    @Override
    public void setY(int y) {
        super.setY(y);
        this.repositionBody();
    }

    @Override
    public boolean mouseScrolled(
        double mouseX,
        double mouseY,
        double deltaX,
        double deltaY
    ) {
        if (!this.isMouseOver(mouseX, mouseY) || this.maxScrollAmount() <= 0) {
            return false;
        }
        this.setScrollAmount(
            this.scrollAmount - deltaY * SCROLL_STEP
        );
        return true;
    }

    @Override
    public boolean mouseClicked(
        double mouseX,
        double mouseY,
        int button
    ) {
        if (
            button == 0 &&
            this.scrollbarVisible() &&
            this.isInScrollbar(mouseX, mouseY)
        ) {
            this.draggingScrollbar = true;
            this.setFocused(true);
            this.setScrollFromMouse(mouseY);
            return true;
        }
        return super.mouseClicked(mouseX, mouseY, button);
    }

    @Override
    public boolean mouseDragged(
        double mouseX,
        double mouseY,
        int button,
        double dragX,
        double dragY
    ) {
        if (button == 0 && this.draggingScrollbar) {
            this.setScrollFromMouse(mouseY);
            return true;
        }
        return super.mouseDragged(
            mouseX,
            mouseY,
            button,
            dragX,
            dragY
        );
    }

    @Override
    public boolean mouseReleased(
        double mouseX,
        double mouseY,
        int button
    ) {
        if (button == 0 && this.draggingScrollbar) {
            this.draggingScrollbar = false;
            return true;
        }
        return super.mouseReleased(mouseX, mouseY, button);
    }

    @Override
    protected void renderWidget(
        GuiGraphics graphics,
        int mouseX,
        int mouseY,
        float partialTick
    ) {
        graphics.enableScissor(
            this.getX(),
            this.getY(),
            this.getX() + this.getWidth() - SCROLLBAR_WIDTH,
            this.getY() + this.getHeight()
        );
        for (AbstractWidget child : this.children) {
            if (
                child.getBottom() >= this.getY() &&
                child.getY() <= this.getBottom()
            ) {
                child.render(graphics, mouseX, mouseY, partialTick);
            }
        }
        graphics.disableScissor();

        if (this.scrollbarVisible()) {
            int trackX = this.getX() + this.getWidth() - SCROLLBAR_WIDTH;
            int thumbHeight = this.scrollbarThumbHeight();
            int travel = Math.max(1, this.getHeight() - thumbHeight);
            int thumbY = this.getY() +
                (int)Math.round(
                    (this.scrollAmount / this.maxScrollAmount()) * travel
                );
            graphics.fill(
                trackX + 2,
                this.getY(),
                trackX + SCROLLBAR_WIDTH - 1,
                this.getBottom(),
                0x44000000
            );
            graphics.fill(
                trackX + 1,
                thumbY,
                trackX + SCROLLBAR_WIDTH,
                thumbY + thumbHeight,
                0xFF8A8A8A
            );
        }
    }

    @Override
    protected void updateWidgetNarration(
        NarrationElementOutput output
    ) {
        output.add(
            NarratedElementType.TITLE,
            this.getMessage()
        );
    }

    private void repositionBody() {
        if (this.body == null) {
            return;
        }
        int bodyX = this.getX();
        int bodyY = this.getY() - (int)Math.round(this.scrollAmount);
        this.body.setX(bodyX);
        this.body.setY(bodyY);
    }

    private int maxScrollAmount() {
        return Math.max(0, this.body.getHeight() - this.getHeight());
    }

    private boolean scrollbarVisible() {
        return this.maxScrollAmount() > 0;
    }

    private int scrollbarThumbHeight() {
        if (!this.scrollbarVisible()) {
            return this.getHeight();
        }
        return Mth.clamp(
            (int)Math.round(
                (double)this.getHeight() * this.getHeight() /
                this.body.getHeight()
            ),
            24,
            this.getHeight()
        );
    }

    private boolean isInScrollbar(double mouseX, double mouseY) {
        int left = this.getX() + this.getWidth() - SCROLLBAR_WIDTH;
        return mouseX >= left &&
            mouseX <= this.getRight() &&
            mouseY >= this.getY() &&
            mouseY <= this.getBottom();
    }

    private void setScrollFromMouse(double mouseY) {
        int thumbHeight = this.scrollbarThumbHeight();
        int travel = Math.max(1, this.getHeight() - thumbHeight);
        double fraction = Mth.clamp(
            (mouseY - this.getY() - thumbHeight / 2.0) / travel,
            0.0,
            1.0
        );
        this.setScrollAmount(fraction * this.maxScrollAmount());
    }

    private void setScrollAmount(double amount) {
        this.scrollAmount = Mth.clamp(
            amount,
            0.0,
            (double)this.maxScrollAmount()
        );
        this.repositionBody();
    }
}
